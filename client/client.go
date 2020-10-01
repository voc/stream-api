package client

import (
	"log"
	"time"
	"fmt"
	"strings"
	"context"
	"sort"

	"go.etcd.io/etcd/clientv3"
	"bitbucket.fem.tu-ilmenau.de/scm/~ischluff/stream-api/config"
	"bitbucket.fem.tu-ilmenau.de/scm/~ischluff/stream-api/service"
)

const requestTimeout := time.Second

// sanitizeKey replaces reserved characters with underscores
func sanitizeKey(key string) string {
	return strings.Replace(key, ":", "_", -1)
}

// Client is a wrapper for a etcdv3 client
type Client struct {
	db *clientv3.Client
	lease clientv3.LeaseID
	name string
}

func NewClient(ctx context.Context, cfg config.Network) *Client {
	c, err := clientv3.New(clientv3.Config{
		Endpoints: cfg.Endpoints,
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
				log.Println("client close")
				return
			case res := <- keepalive:
				// todo: renew lease if res == nil
				log.Println("keepalive", res)
			}
		}
	}()

	return &Client{
		client: c,
		lease: resp.ID,
		name: config.Name
	}
}

func (client *Client) Publish(ctx context.Context, services []service.Service) {
	for _, svc := range services {
		// if its a source watch for updates
		if source, ok := svc.(service.Source); ok {
			watchSource(ctx, source)
		}
		watchService(ctx, svc)
	}
}

func (client *Client) watchSource(ctx context.Context, source service.Source) {
	go func() {
		watch := source.WatchStreams(ctx)
		for {
			select {
			case <-ctx.Done:
				return
			case stream := <-watch:
				publishStream(ctx, stream)
			}
		}
	}
}

func (client *Client) watchService(ctx context.Context, svc service.Service) {
	watch := svc.WatchService(ctx)
	
}

// publishStream publishes a new stream if the key doesn't exist yet
func (client *Client) publishStream(ctx context.Context, stream service.Stream) {
	_, err := client.db.Put(ctx, "stream:%s", stream.Vars(), clientv3.WithLease(client.lease))
	if err != nil {
		log.Fatal(err)
	}
}

// publishService publishes a service endpoint on the current host
func (client *Client) publishService(svc service.Service) {
	key := prefix := fmt.Sprintf("service:%s:%s", svc.Name(), client.Name)
	_, err := client.db.Put(ctx, key, svc.Vars(), clientv3.WithLease(client.lease))
	if err != nil {
		log.Fatal(err)
	}
}

// GetServices fetches all service owners for a given service name
func (client *Client) getServices(c context.Context, service string, rev int64) ([]Service, error) {
	prefix := fmt.Sprintf("service:%s:", service)
	ctx, cancel := context.WithTimeout(c, requestTimeout)
	resp, err := client.db.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithRev(rev))
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	for _, ev := range resp.Kvs {
		fmt.Printf("%s: %s\n", ev.Key, ev.Value)
	}
}

// shouldClaim computes whether we should claim a slot for a certain service
func (client *Client) shouldClaim(name string, ourhost string) bool {
	services, err := client.db.getServices(name)
	if err != nil {
		log.Println(err)
		return false
	}

	sort.Sort(service.ByLoad(services))

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

// ClaimSlot tries to claim a stream slot for a local plugin
func (client *Client) claimSlot(c context.Context, stream Stream, service string) (*Claim, error) {
	key := fmt.Sprintf("stream:%s:claim:%s", sanitizeKey(stream.Name), sanitizeKey(service))

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)

	_, err = kvc.Txn(ctx).
		// Only claim slot if it wasn't claimed before
		// TODO: check if this works, if not do Get first and compare Modversion in Txn
		If(clientv3.Compare(clientv3.Value(key), "!=", "")).
		// Claim for ourselves
		Then(clientv3.OpPut(key, client.hostname, clientv3.WithLease(client.lease))).
		Commit()
	cancel()
	if err != nil {
		return nil, err
	}

	for _, ev := range resp.Kvs {
		fmt.Printf("%s: %s\n", ev.Key, ev.Value)
	}

	return &Claim{
		client: client,
		stream: Stream,
		service: service
	}, _
}

func (client *Client) Watch(ctx) {
	watch := client.Watch("stream/")
	go func(){
		select {
		case <-ctx.Done():
			return
			client.StopWatch(/*TODO*/)

		case stream := <-watch:
			slots := stream.OpenSlots()
			for _, name := range slots {
				plugin, found := reg.Lookup(name)
				if !found {
					continue
				}

				if !shouldClaim(name, config.Hostname) {
					continue
				}

				// see if we should assign ourselves
				// check local capacity
				if plugin.Capacity() - plugin.Active() <= 0 {
					log.Println("Full capacity reached")
					continue
				}

				// try to assign ourselves
				claim, err := client.claimSlot(stream, name)
				if err != nil {
					log.Println(err)
					continue
				}

				if claim != nil {
					service.NewPluginAdaptor(claim, config, plugin, stream)
				}
			}
		}
	}()
}



// Testing
// func (client *Client) Watch(ctx context.Context) {
// 	log.Println("watch")
// 	rch := client.db.Watch(ctx, "stream:", clientv3.WithPrefix())
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

// func (client *Client) Write(ctx context.Context) {
// 	log.Println("write")
// 	_, err := client.db.Put(ctx, "stream:s1:transcode", "bar", clientv3.WithLease(client.lease))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }