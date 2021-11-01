package client

import (
	"context"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/rs/zerolog/log"
	"github.com/voc/stream-api/config"
)

type ConsulClient struct {
	client    *api.Client
	sessionId string
}

func NewConsulClient(parentContext context.Context, cfg config.Network) (*ConsulClient, error) {
	client, err := api.NewClient(&api.Config{})
	if err != nil {
		return nil, err
	}
	session := client.Session()
	id, _, err := session.Create(&api.SessionEntry{
		Name:      cfg.Name,
		Behavior:  "delete",
		LockDelay: time.Second * 5,
		TTL:       "10s",
	}, nil)
	if err != nil {
		return nil, err
	}

	return &ConsulClient{client, id}, nil
}

// PublishWithLease implements PublishAPI
// func (cc *ConsulClient) PublishWithLease(ctx context.Context, key string, value string, ttl time.Duration) (LeaseID, error) {

// }

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

// PublishService implements PublishAPI
func (cc *ConsulClient) PublishService(parentCtx context.Context, service string, data string) error {
	// opts, cancel := optsWithTimeout(parentCtx, time.Second)
	// defer cancel()
	// cc.client.Agent().ServiceRegister(&api.AgentServiceRegistration{
	// 	Name: "publish",

	// })
}

// type PublishAPI interface {
// 	PublishService(ctx context.Context, service string, data string) error
// 	PublishWithLease(ctx context.Context, key string, value string, ttl time.Duration) (LeaseID, error)
// }
