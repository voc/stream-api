package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/coreos/etcd/pkg/transport"
	"github.com/voc/stream-api/config"
	"go.etcd.io/etcd/clientv3"
)

type LeaseID clientv3.LeaseID

// sanitizeKey replaces reserved characters with underscores
func sanitizeKey(key string) string {
	return strings.Replace(key, ":", "_", -1)
}

// Client is a wrapper for a etcdv3 client
type Client struct {
	client *clientv3.Client
	lease  clientv3.LeaseID
	done   sync.WaitGroup
	name   string
	err    chan error
}

func NewEtcdClient(parentContext context.Context, cfg config.Network) *Client {
	var tlsConfig *tls.Config
	if cfg.TLS != nil {
		tlsInfo := transport.TLSInfo{
			CertFile:      cfg.TLS.CertFile,
			KeyFile:       cfg.TLS.KeyFile,
			TrustedCAFile: cfg.TLS.TrustedCAFile,
		}
		var err error
		tlsConfig, err = tlsInfo.ClientConfig()
		if err != nil {
			log.Fatal().Err(err).Msg("client: tls")
		}
	}
	log.Info().Msg("client: connecting to etcd")
	c, err := clientv3.New(clientv3.Config{
		Endpoints: cfg.Endpoints,
		TLS:       tlsConfig,
		// DialKeepAliveTime: time.Second * 5,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("client: new")
	}
	log.Info().Msgf("client: waiting for lease")

	// minimum lease TTL is 5-second
	resp, err := c.Grant(parentContext, 10)
	if err != nil {
		log.Fatal().Err(err).Msg("grant")
	}

	cli := &Client{
		client: c,
		lease:  resp.ID,
		name:   cfg.Name,
		err:    make(chan error),
	}

	// keepalive and revoke lease on exit
	cli.done.Add(1)
	go func() {
		defer cli.done.Done()
		cli.run(parentContext)
	}()

	return cli
}

func (client *Client) Errors() <-chan error {
	return client.err
}

func (client *Client) run(ctx context.Context) {
	keepalive, err := client.client.KeepAlive(ctx, client.lease)
	if err != nil {
		log.Fatal().Err(err).Msg("client: keepalive")
	}
	log.Debug().Msg("client: running keepalive")
	err = client.keepalive(ctx, keepalive)
	if err != nil {
		client.err <- err
	}
}

func (client *Client) keepalive(ctx context.Context, keepalive <-chan *clientv3.LeaseKeepAliveResponse) error {
	for {
		select {
		case <-ctx.Done():
			ctx2, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			_, err := client.client.Revoke(ctx2, client.lease)
			if err != nil {
				log.Error().Err(err).Msg("client: revoke")
			}
			err = client.client.Close()
			if err != nil {
				log.Error().Err(err).Msg("client: close")
			}
			return nil
		case _, ok := <-keepalive:
			if !ok {
				return errors.New("keepalive stopped")
			}
		}
	}
}

func (client *Client) Wait() {
	client.done.Wait()
}

// publishWithLease publishes a key with a new lease if the key doesn't exist yet
func (client *Client) PublishWithLease(ctx context.Context, key string, value string, ttl time.Duration) (LeaseID, error) {
	resp, err := client.client.Grant(ctx, int64(ttl/time.Second))
	if err != nil {
		return 0, err
	}

	res, err := client.client.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, value, clientv3.WithLease(resp.ID))).
		Commit()

	if err != nil {
		return 0, err
	}

	if !res.Succeeded {
		return 0, fmt.Errorf("key %s already exists", key)
	}

	return LeaseID(resp.ID), nil
}

const (
	UpdateTypeDelete = clientv3.EventTypeDelete
	UpdateTypePut    = clientv3.EventTypePut
)

type WatchUpdate struct {
	Type mvccpb.Event_EventType
	KV   *mvccpb.KeyValue
}

type UpdateChan chan []*WatchUpdate

// watch prefix and receive current state
func (client *Client) Watch(ctx context.Context, prefix string) (UpdateChan, error) {
	log.Debug().Msgf("client: watching %s", prefix)
	ctx2, cancel := context.WithCancel(clientv3.WithRequireLeader(ctx))

	opts := []clientv3.OpOption{clientv3.WithPrefix(), clientv3.WithProgressNotify()}
	watchChan := client.client.Watch(ctx2, prefix, opts...)

	// get current state
	var updates []*WatchUpdate
	resp, err := client.client.Get(clientv3.WithRequireLeader(ctx), prefix, clientv3.WithPrefix())
	if err != nil {
		cancel()
		return nil, err
	}
	for _, item := range resp.Kvs {
		updates = append(updates, &WatchUpdate{
			Type: UpdateTypePut,
			KV:   item,
		})
	}
	lastRev := resp.Header.Revision
	ch := make(UpdateChan)

	go func() {
		defer cancel()
		for {
			// push update
			if updates != nil {
				select {
				case <-ctx2.Done():
					return
				case ch <- updates:
				}
				updates = nil
			}

			// wait for data
			select {
			case <-ctx2.Done():
				return
			case change := <-watchChan:
				err := change.Err()
				if change.Canceled {
					log.Error().Err(err).Msg("client: watch closed")
					return
				}
				if err != nil {
					log.Error().Err(err).Msg("client: watch")
					continue
				}
				if change.Header.Revision <= lastRev {
					continue
				}
				for _, event := range change.Events {
					updates = append(updates, &WatchUpdate{
						Type: event.Type,
						KV:   event.Kv,
					})
				}
				lastRev = change.Header.Revision
			}
		}
	}()
	return ch, nil
}

func (client *Client) RefreshLease(ctx context.Context, id LeaseID) error {
	_, err := client.client.KeepAliveOnce(ctx, clientv3.LeaseID(id))
	return err
}

func (client *Client) RevokeLease(ctx context.Context, id LeaseID) error {
	_, err := client.client.Revoke(ctx, clientv3.LeaseID(id))
	return err
}

// publishService publishes a service endpoint on the current host
func (client *Client) PublishService(ctx context.Context, service string, data string) error {
	key := ServicePath(service, client.name)
	log.Info().Msgf("client/publish %s", key)
	// TODO: implement as transaction to prevent double publish!
	_, err := client.client.Put(ctx, key, data, clientv3.WithLease(client.lease))
	return err
}

func (client *Client) Put(ctx context.Context, key string, value []byte) error {
	_, err := client.client.KV.Put(ctx, key, string(value))
	return err
}

func (client *Client) Get(ctx context.Context, key string) ([]byte, error) {
	res, err := client.client.KV.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(res.Kvs) == 0 {
		return nil, errors.New("empty")
	}

	return res.Kvs[0].Value, nil
}

func (client *Client) GetWithPrefix(ctx context.Context, prefix string) ([]Field, error) {
	res, err := client.client.KV.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	var fields []Field
	// fmt.Println("huhu", res.Kvs)
	for _, kv := range res.Kvs {
		fields = append(fields, Field{
			Key:   kv.Key,
			Value: kv.Value,
		})
	}
	return fields, nil
}

// GetServices fetches all service owners for a given service name
// func (client *Client) getServices(c context.Context, service string, rev int64) ([]service.Service, error) {
// 	prefix := fmt.Sprintf("service:%s:", service)
// 	ctx, cancel := context.WithTimeout(c, requestTimeout)
// 	resp, err := client.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithRev(rev))
// 	cancel()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	for _, ev := range resp.Kvs {
// 		fmt.Printf("%s: %s\n", ev.Key, ev.Value)
// 	}
// 	return nil, nil // TODO
// }
