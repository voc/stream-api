package client

import (
	"context"
)

type UpdateType int

const (
	UpdateTypePut UpdateType = iota
	UpdateTypeDelete
)

type KeyValue interface {
	Key() string
	Value() []byte
}

type WatchUpdate struct {
	Type UpdateType
	KV   KeyValue
}

type UpdateChan chan []*WatchUpdate

type WatchAPI interface {
	// Watch Prefix
	Watch(ctx context.Context, prefix string) (UpdateChan, error)
}

// type PublishAPI interface {
// 	PublishService(ctx context.Context, service string, data string) error
// 	PublishWithLease(ctx context.Context, key string, value string, ttl time.Duration) (LeaseID, error)
// PublishWithSession(ctx context.Context, key string, value []byte) error
// }

type Field struct {
	Key   []byte
	Value []byte
}

type KVAPI interface {
	// Get(ctx context.Context, key string) ([]byte, error)
	// GetWithPrefix(ctx context.Context, prefix string) ([]Field, error)
	Put(ctx context.Context, key string, value []byte) error
	PutWithSession(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
}

type KeepaliveAPI interface {
	RefreshLease(ctx context.Context, id LeaseID) error
	RevokeLease(ctx context.Context, id LeaseID) error
}

type ServiceAPI interface {
	WatchAPI
	// PublishAPI
	// KeepaliveAPI
	KVAPI
}
