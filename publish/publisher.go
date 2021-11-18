package publish

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/voc/stream-api/client"
	"github.com/voc/stream-api/config"
	"github.com/voc/stream-api/publish/source"
	"github.com/voc/stream-api/stream"
)

type storedStream struct {
	st  *stream.Stream
	ttl int
}

// Publisher publishes streams to the etcd store and keeps them refreshed
type Publisher struct {
	conf    *config.PublisherConfig
	ttl     int
	streams map[string]*storedStream
	update  chan struct{}
	name    string
	api     client.ServiceAPI
	done    sync.WaitGroup
}

var defaultScrapeInterval = time.Second * 3

// New creates a new Publisher
func New(ctx context.Context, conf *config.PublisherConfig, api client.ServiceAPI, name string) *Publisher {
	// create stream publishers
	var scrapers []source.Scraper
	for _, sourceConfig := range conf.Sources {
		switch sourceConfig.Type {
		case "icecast":
			scrapers = append(scrapers, source.NewIcecastScraper(sourceConfig))
		case "srtrelay":
			scrapers = append(scrapers, source.NewSrtrelayScraper(sourceConfig))
		default:
			log.Error().Msgf("publisher: unknown source type %s", sourceConfig.Type)
		}
	}

	p := &Publisher{
		conf:    conf,
		ttl:     int(conf.Timeout / conf.Interval),
		update:  make(chan struct{}),
		streams: make(map[string]*storedStream),
		name:    name,
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
					log.Error().Err(err).Msg("publisher/scrape")
					continue
				}
				p.processUpdate(ctx, streams)
			}

			// expire old streams
			for key, stream := range p.streams {
				stream.ttl--
				if stream.ttl > 0 {
					continue
				}
				err := p.unpublishStream(ctx, stream)
				if err != nil {
					log.Error().Err(err).Msg("publisher/unpublish")
					continue
				}
				delete(p.streams, key)
			}
		}
	}
}

func (p *Publisher) unpublishStream(ctx context.Context, stream *storedStream) error {
	key := client.StreamPath(stream.st.Slug)
	log.Debug().Str("slug", stream.st.Slug).Msg("publisher/unpublish")
	return p.api.Delete(ctx, key)
}

func (p *Publisher) publishStream(ctx context.Context, stream *stream.Stream) error {
	key := client.StreamPath(stream.Slug)
	val, err := json.Marshal(stream)
	if err != nil {
		return err
	}
	return p.api.PutWithSession(ctx, key, val)
}

// TODO: handle local updates to stream data (e.g. more than one source with the same slug -> do a flat comparison)
func (p *Publisher) processUpdate(ctx context.Context, streams []*stream.Stream) {
	var newStreams []*storedStream
	for _, stream := range streams {
		stored, exists := p.streams[stream.Slug]
		if exists {
			// renew timeout
			stored.ttl = p.ttl
		}
		err := p.publishStream(ctx, stream)
		if err != nil {
			log.Error().Str("slug", stream.Slug).Err(err).Msg("publisher/publish")
			continue
		}
		if !exists {
			log.Debug().Str("slug", stream.Slug).Str("source", stream.Source).Err(err).Msg("publisher/publish")
			newStreams = append(newStreams, &storedStream{
				st:  stream,
				ttl: p.ttl,
			})
			continue
		}

	}

	// add new streams
	for _, s := range newStreams {
		p.streams[s.st.Slug] = s
	}
}
