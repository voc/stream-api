package fanout

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/voc/stream-api/client"
	"github.com/voc/stream-api/config"
	"github.com/voc/stream-api/stream"
	"github.com/voc/stream-api/systemd"
)

var fanoutTTL = 10 * time.Second

type Fanout struct {
	api        client.ServiceAPI
	done       sync.WaitGroup
	name       string
	configPath string
	sink       string

	// local state
	services map[string]*systemd.Service
	streams  map[string]*stream.Stream
}

func New(ctx context.Context, conf config.FanoutConfig, api client.ServiceAPI, name string) *Fanout {
	log.Debug().Msgf("fanout config %v", conf)
	t := &Fanout{
		api:        api,
		services:   make(map[string]*systemd.Service),
		streams:    make(map[string]*stream.Stream),
		name:       name,
		configPath: conf.ConfigPath,
		sink:       conf.Sink,
	}

	// watch source updates
	t.done.Add(1)
	go t.run(ctx)

	return t
}

func (t *Fanout) Wait() {
	for _, service := range t.services {
		service.Wait()
	}
	t.done.Wait()
}

// run keeps the communication to etcd
func (t *Fanout) run(parentContext context.Context) {
	defer t.done.Done()
	ctx, cancel := context.WithCancel(parentContext)
	defer cancel()

	t.publishStatus(ctx)
	streamChan, err := t.api.Watch(ctx, "stream:")
	if err != nil {
		log.Fatal().Err(err).Msg("stream watch")
		return
	}
	ticker := time.NewTicker(fanoutTTL)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case updates, ok := <-streamChan:
			if !ok {
				log.Fatal().Msg("stream watch closed")
				return
			}
			for _, update := range updates {
				t.handleStream(ctx, update)
			}
		// perform periodic updates
		case <-ticker.C:
			for key, service := range t.services {
				// cleanup stopped services
				if service.Stopped() {
					log.Info().Msgf("fanout/service: stopped %s", key)
					delete(t.services, key)
				}
				// stop unnecessary services
				if _, found := t.streams[key]; !found {
					service.Stop()
				}
			}
		}
	}
}

// publishStatus announces the fanout to the network
func (t *Fanout) publishStatus(ctx context.Context) {
	var streams []string
	for key := range t.services {
		streams = append(streams, key)
	}
	status := &FanoutStatus{
		Name:    t.name,
		Sink:    t.sink,
		Streams: streams,
	}
	data, err := json.Marshal(status)
	if err != nil {
		log.Error().Err(err).Msg("fanout marshal")
		return
	}
	t.api.PublishService(ctx, "fanout", string(data))
}

// handleStream handles an update in the etcd stream prefix
func (t *Fanout) handleStream(ctx context.Context, update *client.WatchUpdate) {
	if update.KV == nil {
		return
	}

	parts := strings.Split(string(update.KV.Key), ":")
	if len(parts) != 2 {
		return
	}
	key := parts[1]

	switch update.Type {
	case client.UpdateTypePut:
		var str stream.Stream
		err := json.Unmarshal(update.KV.Value, &str)
		if err != nil {
			log.Error().Err(err).Msg("stream unmarshal")
			return
		}
		t.streams[key] = &str
		t.claimStream(ctx, &str)

	case client.UpdateTypeDelete:
		delete(t.streams, key)
	}
	log.Debug().Msgf("streams %v", t.streams)
}

// claimStream claims a stream for the current fanout
func (t *Fanout) claimStream(ctx context.Context, s *stream.Stream) {
	service, err := t.createService(ctx, s)
	if err != nil {
		log.Error().Err(err).Msgf("fanout/claim: service %s", s.Slug)
	}
	t.services[s.Slug] = service
	t.publishStatus(ctx)
}

var configTemplate = template.Must(template.New("fanoutConfig").Parse(`
stream_key={{ .Slug }}
fanout_source={{ .Source }}
`))

func (t *Fanout) createService(ctx context.Context, s *stream.Stream) (*systemd.Service, error) {
	type StreamConfig struct {
		Slug   string
		Source string
	}
	var buf bytes.Buffer
	err := configTemplate.Execute(&buf, &StreamConfig{
		Slug:   s.Slug,
		Source: t.sink,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("fanout: templateConfig")
	}

	return systemd.NewService(ctx, &systemd.ServiceConfig{
		Config:     buf.Bytes(),
		ConfigPath: path.Join(t.configPath, s.Slug),
		UnitName:   fmt.Sprintf("fanout@%s.target", s.Slug),
	})
}
