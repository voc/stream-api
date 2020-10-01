package db

import (
	"log"
	"time"
	"fmt"
	"strings"
	"context"

	"go.etcd.io/etcd/clientv3"
)

const requestTimeout := time.Second

// sanitizeKey replaces reserved characters with underscores
func sanitizeKey(key string) string {
	return strings.Replace(key, ":", "_", -1)
}

type Claim struct {
	db *DB
}

func (cl *Claim) Update(Vars interface{}) {
	_, err := db.client.Put(ctx, "stream:s1:transcode", "bar", clientv3.WithLease(db.lease))
	if err != nil {
		log.Fatal(err)
	}
}

// DB is a wrapper for a etcdv3 client
type DB struct {
	client *clientv3.Client
	lease clientv3.LeaseID
	hostname string
}

func NewDB(ctx context.Context, endpoints []string, hostname string) *DB {
	c, err := clientv3.New(clientv3.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		log.Fatal(err)
	}

	// minimum lease TTL is 5-second
	resp, err := c.Grant(ctx, 5)
	if err != nil {
		log.Fatal(err)
	}

	var keepalive <-chan *clientv3.LeaseKeepAliveResponse
	keepalive, err = c.KeepAlive(ctx, resp.ID)
	if err != nil {
		log.Fatal(err)
	}

	// keepalive and revoke lease on exit
	go func(){
		for {
			select {
			case <-ctx.Done():
				ctx2, _ := context.WithTimeout(context.Background(), 300*time.Millisecond)
				_, err = c.Revoke(ctx2, resp.ID)
				if err != nil {
				    log.Println(err)
				}
				c.Close()
				log.Println("db close")
				return
			case renew := <- keepalive:
				log.Println("keepalive", renew)
			}
		}
	}()

	return &DB{
		client: c,
		lease: resp.ID,
		hostname: hostname,
	}
}


// GetServices fetches all service owners for a given service name
func (db *DB) GetServices(c context.Context, service string, rev int64) ([]Service, error) {
	prefix := fmt.Sprintf("service:%s:", service)
	ctx, cancel := context.WithTimeout(c, requestTimeout)
	resp, err := db.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithRev(rev))
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	for _, ev := range resp.Kvs {
		fmt.Printf("%s: %s\n", ev.Key, ev.Value)
	}
}

// ClaimSlot tries to claim a stream slot for a local plugin
func (db *DB) ClaimSlot(c context.Context, stream Stream, service string) (*Claim, error) {
	key := fmt.Sprintf("stream:%s:claim:%s", sanitizeKey(stream.Name), sanitizeKey(service))

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)

	_, err = kvc.Txn(ctx).
		// Only claim slot if it wasn't claimed before
		// TODO: check if this works, if not do Get first and compare Modversion in Txn
	    If(clientv3.Compare(clientv3.Value(key), "!=", "")).
	    // Claim for ourselves
	    Then(clientv3.OpPut(key, db.hostname, clientv3.WithLease(db.lease))).
	    Commit()
	cancel()
	if err != nil {
		return nil, err
	}

	for _, ev := range resp.Kvs {
		fmt.Printf("%s: %s\n", ev.Key, ev.Value)
	}

	return &Claim{
		db: db
	}, _
}

func (db *DB) Watch(ctx) {
	watch := db.Watch("stream/")
	go func(){
		select {
		case <-ctx.Done():
			return
			db.StopWatch(/*TODO*/)

		case stream := <-watch:
			slots := stream.OpenSlots()
			for _, name := range slots {
				plugin, found := reg.Lookup(name)
				if !found {
					continue
				}

				if !ShouldClaim(name, config.Hostname) {
					continue
				}

				// see if we should assign ourselves
				// check local capacity
				if plugin.Capacity() - plugin.Active() <= 0 {
					log.Println("Full capacity reached")
					continue
				}

				// try to assign ourselves
				claim, err := db.ClaimSlot(stream, name)
				if err != nil {
					log.Println(err)
					continue
				}

				if claim != nil {
					NewPluginAdaptor(claim, config, plugin, stream)
				}
			}
		}
	}()
}



// Testing
// func (db *DB) Watch(ctx context.Context) {
// 	log.Println("watch")
// 	rch := db.client.Watch(ctx, "stream:", clientv3.WithPrefix())
// 	go func(){
// 		for {
// 			select {
// 			case <-ctx.Done():
// 				log.Println("watch exit")
// 				return
// 			case wresp := <- rch:
// 				for _, ev := range wresp.Events {
// 			        fmt.Printf("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
// 			        fmt.Printf("wresp.Header.Revision: %d\n", wresp.Header.Revision)
// 					fmt.Println("wresp.IsProgressNotify:", wresp.IsProgressNotify())
// 			    }
// 			}
// 		}
// 	}()
// }

// func (db *DB) Write(ctx context.Context) {
// 	log.Println("write")
// 	_, err := db.client.Put(ctx, "stream:s1:transcode", "bar", clientv3.WithLease(db.lease))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }