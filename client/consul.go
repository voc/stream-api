package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/rs/zerolog/log"
	"github.com/voc/stream-api/config"
)

type ConsulClient struct {
	client     *api.Client
	conf       config.Network
	sessionTTL string
	sessionId  string
	done       sync.WaitGroup
}

func NewConsulClient(parentContext context.Context, conf config.Network) (*ConsulClient, error) {
	client, err := api.NewClient(&api.Config{})
	if err != nil {
		return nil, err
	}
	cc := &ConsulClient{client: client, conf: conf, sessionTTL: "10s"}
	err = cc.renewSession()
	if err != nil {
		return nil, err
	}
	cc.done.Add(1)
	go cc.keepaliveSession(parentContext)
	return cc, nil
}

func (cc *ConsulClient) renewSession() error {
	session := cc.client.Session()
	id, _, err := session.Create(&api.SessionEntry{
		Name:      cc.conf.Name,
		Behavior:  "delete",
		LockDelay: time.Second * 5,
		TTL:       cc.sessionTTL,
	}, nil)
	if err != nil {
		return err
	}
	cc.sessionId = id
	return nil
}

func (cc *ConsulClient) keepaliveSession(ctx context.Context) {
	session := cc.client.Session()
	for {
		err := session.RenewPeriodic(cc.sessionTTL, cc.sessionId, nil, ctx.Done())
		log.Warn().Err(err).Msg("failed to renew session")
		time.Sleep(time.Second * 3)
		cc.renewSession()
	}
}

func (cc *ConsulClient) Errors() <-chan error {
	return make(chan error)
}

type ConsulKV struct {
	kv *api.KVPair
}

func (c *ConsulKV) Key() string {
	return c.kv.Key
}

func (c *ConsulKV) Value() []byte {
	return c.kv.Value
}

// PublishWithLease implements PublishAPI
// func (cc *ConsulClient) PublishWithLease(ctx context.Context, key string, value string, ttl time.Duration) (LeaseID, error) {

// }

// helper method
func optsWithTimeout(parentCtx context.Context, timeout time.Duration) (*api.WriteOptions, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	opts := &api.WriteOptions{}
	return opts.WithContext(ctx), cancel
}

func (cc *ConsulClient) Close() {
	// destroy session
	session := cc.client.Session()
	opts, cancel := optsWithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := session.Destroy(cc.sessionId, opts)
	if err != nil {
		log.Error().Err(err).Msg("destroy session")
	}
}

// Watch watches the consul key for changes
func (cc *ConsulClient) Watch(ctx context.Context, prefix string) (UpdateChan, error) {
	query := map[string]interface{}{
		"type":   "keyprefix",
		"prefix": prefix,
	}
	plan, err := watch.Parse(query)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("prefix", prefix).Msg("watch")
	ch := make(UpdateChan)
	plan.HybridHandler = cc.makeWatchHandler(ch)

	// run plan
	go func() {
		err := plan.RunWithClientAndHclog(cc.client, nil)
		log.Error().Err(err).Msg("watch stopped")
	}()

	// stop plan
	go func() {
		<-ctx.Done()
		plan.Stop()
	}()
	return ch, nil
}

// handleWatch updates the cache on consul changes
func (cc *ConsulClient) makeWatchHandler(ch UpdateChan) watch.HybridHandlerFunc {
	return func(b watch.BlockingParamVal, update interface{}) {
		switch val := update.(type) {
		case *api.KVPair:
			if val == nil {
				return
			}
			ch <- []*WatchUpdate{{
				KV: &ConsulKV{kv: val},
			}}
		case api.KVPairs:
			update := make([]*WatchUpdate, 0, len(val))
			for _, pair := range val {
				update = append(update, &WatchUpdate{
					KV: &ConsulKV{kv: pair},
				})
			}
			ch <- update
		default:
			log.Error().Msg("watch: invalid update")
		}
	}
}

// kv put
func (cc *ConsulClient) Put(ctx context.Context, key string, value []byte) error {
	p := &api.KVPair{Key: key, Value: value}
	opts, cancel := optsWithTimeout(ctx, time.Second)
	defer cancel()
	_, err := cc.client.KV().Put(p, opts)
	return err
}

// kv put with expiring session
type ErrAlreadyAquired struct {
	Key string
}

func (e *ErrAlreadyAquired) Error() string {
	return fmt.Sprintf("key %s already aquired", e.Key)
}

func (cc *ConsulClient) PutWithSession(ctx context.Context, key string, value []byte) error {
	p := &api.KVPair{Key: key, Value: value, Session: cc.sessionId}
	opts, cancel := optsWithTimeout(ctx, time.Second)
	defer cancel()
	success, _, err := cc.client.KV().Acquire(p, opts)
	if !success {
		if err != nil {
			return fmt.Errorf("acquire failed: %w", err)
		}
		return &ErrAlreadyAquired{
			Key: key,
		}
	}
	return err
}

// kv delete with expiring session
func (cc *ConsulClient) Delete(ctx context.Context, key string) error {
	opts, cancel := optsWithTimeout(ctx, time.Second)
	defer cancel()
	_, err := cc.client.KV().Delete(key, opts)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}
	return err
}

// PublishService implements PublishAPI
// func (cc *ConsulClient) PublishService(parentCtx context.Context, service string, data string) error {
// opts, cancel := optsWithTimeout(parentCtx, time.Second)
// defer cancel()
// err := cc.client.Agent().ServiceRegister(&api.AgentServiceRegistration{
// 	Name: service,
// 	Port: ,
// })
// return err
// }

// type PublishAPI interface {
// 	PublishService(ctx context.Context, service string, data string) error
// 	PublishWithLease(ctx context.Context, key string, value string, ttl time.Duration) (LeaseID, error)
// }
