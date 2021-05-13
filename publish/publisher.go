package publish

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/voc/stream-api/client"
	"github.com/voc/stream-api/config"
	"github.com/voc/stream-api/source"
	"github.com/voc/stream-api/stream"
)

type storedStream struct {
	stream *stream.Stream
	lease  client.LeaseID
}

// Publisher publishes streams to the etcd store and keeps them refreshed
type Publisher struct {
	streams map[string]*storedStream
	update  chan struct{}
	api     client.PublisherAPI
	done    sync.WaitGroup
}

var defaultScrapeInterval = time.Second * 3

// NewPublisher creates a new Publisher
func NewPublisher(ctx context.Context, api client.PublisherAPI, configs []config.SourceConfig) *Publisher {
	// create stream publishers
	var scrapers []source.Scraper
	for _, scraperConfig := range configs {
		switch scraperConfig.Type {
		case "icecast":
			scrapers = append(scrapers, source.NewIcecastScraper(scraperConfig))
		case "srtrelay":
			scrapers = append(scrapers, source.NewSrtrelayScraper(scraperConfig))
		default:
			log.Println("Invalid publisher type:", scraperConfig.Type)
		}
	}

	p := &Publisher{
		update:  make(chan struct{}),
		streams: make(map[string]*storedStream),
		api:     api,
	}

	// watch source updates
	p.done.Add(1)
	go p.run(ctx, scrapers)

	return p
}

func (p *Publisher) Wait() {
	p.done.Wait()
}

func (p *Publisher) run(parentContext context.Context, scrapers []source.Scraper) {
	ctx, cancel := context.WithCancel(parentContext)
	defer cancel()

	watchChan, err := p.api.Watch(ctx, "stream:")
	if err != nil {
		log.Println("publish/watch:", err.Error())
		return
	}

	ticker := time.NewTicker(defaultScrapeInterval)
	defer p.done.Done()
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			for _, scraper := range scrapers {
				timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
				streams, err := scraper.Scrape(timeoutCtx)
				cancel()
				if err != nil {
					log.Println("scrape:", err.Error())
					continue
				}
				p.processUpdate(ctx, streams)
			}
		case updates, ok := <-watchChan:
			if !ok {
				log.Println("update/watch: closed")
				return
			}
			// Delete expired streams from local cache
			for _, update := range updates {
				parts := strings.Split(string(update.KV.Key), ":")
				if len(parts) != 2 {
					continue
				}
				key := parts[1]
				_, exists := p.streams[key]
				if exists && update.Type == client.UpdateTypeDelete {
					delete(p.streams, key)
				}
			}
		}
	}
}

func (p *Publisher) publishStream(ctx context.Context, stream *stream.Stream) (client.LeaseID, error) {
	key := fmt.Sprintf("stream:%s", stream.Slug)
	val, err := json.Marshal(stream)
	if err != nil {
		return 0, err
	}
	return p.api.PublishWithLease(ctx, key, string(val), defaultScrapeInterval*2)
}

// TODO: handle local updates to stream data (e.g. more than one source with the same slug -> do a flat comparison)
func (p *Publisher) processUpdate(ctx context.Context, streams []*stream.Stream) {
	var newStreams []*storedStream
	for _, stream := range streams {
		stored, exists := p.streams[stream.Slug]
		if !exists {
			// setup new stream
			lease, err := p.publishStream(ctx, stream)
			if err != nil {
				log.Println("publish/publish:", err)
				continue
			}
			log.Println("publish", stream)
			newStreams = append(newStreams, &storedStream{
				stream: stream,
				lease:  lease,
			})
		} else {
			// refresh stream lease
			err := p.api.RefreshLease(ctx, stored.lease)
			if err != nil {
				log.Println("publish/refresh:", err)
			}
		}
	}

	// add new streams
	for _, s := range newStreams {
		p.streams[s.stream.Slug] = s
	}
}
