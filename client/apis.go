package client

import (
	"context"
	"time"
)

type WatchAPI interface {
	Watch(ctx context.Context, prefix string) (UpdateChan, error)
}

type PublishAPI interface {
	PublishService(ctx context.Context, service string, data string) error
	PublishWithLease(ctx context.Context, key string, value string, ttl time.Duration) (LeaseID, error)
}

type Field struct {
	Key   []byte
	Value []byte
}

type RestAPI interface {
	Get(ctx context.Context, key string) ([]byte, error)
	GetWithPrefix(ctx context.Context, prefix string) ([]Field, error)
	Put(ctx context.Context, key string, value []byte) error
}

type KeepaliveAPI interface {
	RefreshLease(ctx context.Context, id LeaseID) error
	RevokeLease(ctx context.Context, id LeaseID) error
}

type ServiceAPI interface {
	WatchAPI
	PublishAPI
	KeepaliveAPI
	RestAPI
}
