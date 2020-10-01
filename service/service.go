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

type Service struct {
	Host string
	Capacity int
	Active int
}

// ByLoad implements sort.Interface for []Service based on job load
type ByLoad []Service
func (l ByLoad) Len() int { return len(l) }
func (l ByLoad) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (l ByLoad) Less(i, j int) {
	// If we have no capacity order by count alone
	if l[i].Capacity <= 0 or l[j].Capacity <= 0 {
		return l[i].Active < l[j].Active
	}

	// Order based on load
	return float64(l[i].Active) / float64(l[i].Capacity) <
		float64(l[j].Active) / float64(l[j].Capacity)
}


// ShouldClaim computes whether we should claim a slot for a certain service
func ShouldClaim(name string, ourhost string) bool {
	services, err := db.GetServices(name)
	if err != nil {
		log.Println(err)
		return false
	}

	sort.Sort(ByLoad(services))

	// Claim slot if we are part of the top 2 candidates
	n = 2
	if len(services < n) {
		n = len(services)
	}
	max := max()
	for _, s := range services[:n] {
		if s.Host == ourhost {
			return true
		}
	}
	return false
}

type Assigner struct {
	db *DB
}

func NewAssigner(ctx context.Context, db, reg, config Config) *Watcher {

}