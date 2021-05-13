package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/voc/stream-api/config"
	"go.etcd.io/etcd/clientv3"
)

const requestTimeout = time.Second

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
}

func NewClient(parentContext context.Context, cfg config.Network) *Client {
	c, err := clientv3.New(clientv3.Config{
		Endpoints: cfg.Endpoints,
	})
	if err != nil {
		log.Fatal(err)
	}

	// minimum lease TTL is 5-second
	resp, err := c.Grant(parentContext, 5)
	if err != nil {
		log.Fatal(err)
	}

	cli := &Client{
		client: c,
		lease:  resp.ID,
		name:   cfg.Name,
	}

	// keepalive and revoke lease on exit
	cli.done.Add(1)
	go func() {
		defer cli.done.Done()
		cli.run(parentContext)
	}()

	return cli
}

func (client *Client) run(ctx context.Context) {
	for {
		// var keepalive <-chan *clientv3.LeaseKeepAliveResponse
		keepalive, err := client.client.KeepAlive(ctx, client.lease)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("running keepalive")
		done := client.keepalive(ctx, keepalive)
		if done {
			return
		}
	}
}

func (client *Client) keepalive(ctx context.Context, keepalive <-chan *clientv3.LeaseKeepAliveResponse) bool {
	for {
		select {
		case <-ctx.Done():
			ctx2, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			_, err := client.client.Revoke(ctx2, client.lease)
			if err != nil {
				log.Println(err)
			}
			err = client.client.Close()
			if err != nil {
				log.Println("client close:", err.Error())
			}
			return true
		case _, ok := <-keepalive:
			if !ok {
				log.Println("keepalive stopped!")
				return false
			}
		}
	}
}

func (client *Client) Wait() {
	client.done.Wait()
}

type WatchAPI interface {
	Watch(ctx context.Context, prefix string) (UpdateChan, error)
}

type PublishAPI interface {
	PublishService(ctx context.Context, service string, data string) error
	PublishWithLease(ctx context.Context, key string, value string, ttl time.Duration) (LeaseID, error)
}

type KeepaliveAPI interface {
	RefreshLease(ctx context.Context, id LeaseID) error
	RevokeLease(ctx context.Context, id LeaseID) error
}

type ServiceAPI interface {
	WatchAPI
	PublishAPI
	KeepaliveAPI
}

// publishWithLease publishes a key with a new lease if the key doesn't exist yet
func (client *Client) PublishWithLease(ctx context.Context, key string, value string, ttl time.Duration) (LeaseID, error) {
	resp, err := client.client.Grant(ctx, int64(ttl/time.Second))
	if err != nil {
		return 0, err
	}

	res, err := client.client.Txn(ctx).
		// txn value comparisons are lexical
		If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		// the "Then" runs, since "xyz" > "abc"
		Then(clientv3.OpPut(key, value, clientv3.WithLease(resp.ID))).
		Commit()

	if err != nil {
		return 0, err
	}

	if !res.Succeeded {
		return 0, errors.New("already exists")
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
	log.Println("watch", prefix)
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
					log.Println("watch chan closed", err.Error())
					return
				}
				if err != nil {
					log.Println("watch error", err.Error())
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
	key := fmt.Sprintf("service:%s:%s", service, client.name)
	// TODO: implement as transaction to prevent double publish!
	_, err := client.client.Put(ctx, key, data, clientv3.WithLease(client.lease))
	return err
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
