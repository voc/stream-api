package service

import (
	"fmt"
	"sort"
)

// type Source struct {

// }

// type upstream interface {
// 	poll() (*streams, error)
// }

// func NewSource(pub) *Source {
// }

// type Source interface {
// 	Assigner
// 	Service
// 	pubsub - publish/unpublish stream
// 	assigns roles
// }

// type Assigner interface {
// 	Assign (Stream, Resources)
// }

// type Service interface {
// 	GetName()
// 	GetDescription()
// 	GetStreamVars(stream) <-chan (do updates?, probably pubsub)
// 	Start(key, params)
// 	Stop(key)
// }

// Registry is a frontend for the db which handles service registrations and lookups
type Registry struct {
	db *DB
	hostname string
}

func NewRegistry(db *DB, hostname string) *Registry {
	return &Registry{
		db: db,
		hostname: hostname,
	}
}

// Register publishes a service entry
// write static! service registration
func (mng *Registry) Register(service *Service) {
	name := service.GetName()
	db.Write("/services/" + name + "/" + hostname, service.GetDescription())
}

func (mng *Registry) Lookup(string) (*Service, error) {
	//TODO
}

// type Assigner struct {
// 	db *DB
// }

// func NewAssigner(ctx context.Context, db, reg, config Config) *Watcher {

// }

type Service interface {
	Name() string
	Capacity() int
	Active() int
	Add(Stream) error
	Remove(Stream) error
	WatchService() chan -> interface{}
}

type Source interface {
	WatchStreams() chan -> Stream
}

// ByLoad implements sort.Interface for []Service based on job load
type ByLoad []Service
func (l ByLoad) Len() int { return len(l) }
func (l ByLoad) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (l ByLoad) Less(i, j int) {
	// If we have no capacity order by count alone
	if l[i].Capacity() <= 0 or l[j].Capacity() <= 0 {
		return l[i].Active() < l[j].Active()
	}

	// Order based on load
	return float64(l[i].Active()) / float64(l[i].Capacity()) <
		float64(l[j].Active()) / float64(l[j].Capacity())
}