package service

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

type DB struct{}

// Registry is a frontend for the db which handles service registrations and lookups
type Registry struct {
	db       *DB
	hostname string
}

func NewRegistry(db *DB, hostname string) *Registry {
	return &Registry{
		db:       db,
		hostname: hostname,
	}
}

// Register publishes a service entry
// write static! service registration
func (mng *Registry) Register(service Service) {
	// name := service.Name()
	// db.Write("/services/"+name+"/"+hostname, service.GetDescription())
}

func (mng *Registry) Lookup(string) (*Service, error) {
	//TODO
	return nil, nil
}

// type Assigner struct {
// 	db *DB
// }

// func NewAssigner(ctx context.Context, db, reg, config Config) *Watcher {

// }

type Service interface {
	Name() string
	Host()
	Capacity() int
	Active() int
	Add(Stream) error
	Remove(Stream) error
	WatchService() <-chan interface{}
}

type Source interface {
	WatchStreams() <-chan Stream
}
