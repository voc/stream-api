package transcode

// Goals for all transcoders:
//  - distribute streams evenly
//  - when first transcoder is half filled -> assign next
//  - start tracker workers on idle transcoders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
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

var transcoderTTL = 10 * time.Second

type Transcoder struct {
	api        client.TranscoderAPI
	done       sync.WaitGroup
	name       string
	capacity   int
	configPath string
	sink       string // TODO: replace with dynamic discovery

	// local state
	jobs              map[string]*systemd.Service
	transcoders       map[string]*TranscoderStatus
	streams           map[string]*stream.Stream
	streamTranscoders map[string]string
}

func NewTranscoder(ctx context.Context, api client.TranscoderAPI, conf config.TranscodeConfig) *Transcoder {
	fmt.Println("transcoder", conf)
	t := &Transcoder{
		api:               api,
		jobs:              make(map[string]*systemd.Service),
		transcoders:       make(map[string]*TranscoderStatus),
		streams:           make(map[string]*stream.Stream),
		streamTranscoders: make(map[string]string),
		name:              conf.Name,
		capacity:          conf.Capacity,
		configPath:        conf.ConfigPath,
		sink:              conf.Sink,
	}

	// watch source updates
	t.done.Add(1)
	go t.run(ctx)

	return t
}

func (t *Transcoder) Wait() {
	for _, job := range t.jobs {
		job.Wait()
	}
	t.done.Wait()
}

// run keeps the communication to etcd
func (t *Transcoder) run(parentContext context.Context) {
	defer t.done.Done()
	ctx, cancel := context.WithCancel(parentContext)
	defer cancel()

	t.publishStatus(ctx)
	transcoderChan, err := t.api.Watch(ctx, "service:transcoder:")
	if err != nil {
		log.Fatal().Err(err).Msg("transcoder watch")
		return
	}
	streamChan, err := t.api.Watch(ctx, "stream:")
	if err != nil {
		log.Fatal().Err(err).Msg("stream watch")
		return
	}
	ticker := time.NewTicker(transcoderTTL)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case updates, ok := <-transcoderChan:
			if !ok {
				log.Fatal().Msg("transcoder watch closed")
				return
			}
			for _, update := range updates {
				t.handleTranscoder(update)
			}
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
			for key, job := range t.jobs {
				// cleanup stopped jobs
				if job.Stopped() {
					log.Info().Msgf("transcode/job: stopped %s", key)
					delete(t.jobs, key)
				}
				// stop unnecessary jobs
				if _, found := t.streams[key]; !found {
					job.Stop()
				}
			}

			// check whether we have capacity
			if !t.shouldClaim() {
				break
			}
			for key, stream := range t.streams {
				if _, found := t.streamTranscoders[key]; found {
					continue
				}
				// claim streams without active transcoder
				t.claimStream(ctx, stream)

				// check whether we have further capacity
				if !t.shouldClaim() {
					break
				}
			}
		}
	}
}

// publishStatus announces the transcoder to the network
func (t *Transcoder) publishStatus(ctx context.Context) {
	var streams []string
	for key := range t.jobs {
		streams = append(streams, key)
	}
	status := &TranscoderStatus{
		Name:     t.name,
		Capacity: t.capacity,
		Streams:  streams,
	}
	data, err := json.Marshal(status)
	if err != nil {
		log.Error().Err(err).Msg("transcoder marshal")
		return
	}
	t.api.PublishService(ctx, "transcoder", string(data))
}

// handleTranscoder handles an etcd transcoder update
func (t *Transcoder) handleTranscoder(update *client.WatchUpdate) {
	if update.KV == nil {
		return
	}

	parts := strings.Split(string(update.KV.Key), ":")
	if len(parts) != 3 {
		return
	}
	name := parts[2]

	switch update.Type {
	case client.UpdateTypePut:
		var status TranscoderStatus
		err := json.Unmarshal(update.KV.Value, &status)
		if err != nil {
			log.Error().Err(err).Msg("transcoder unmarshal")
			return
		}
		t.transcoders[name] = &status
	case client.UpdateTypeDelete:
		delete(t.transcoders, name)
	}
	log.Debug().Msgf("transcoders %v", t.transcoders)
}

// handleStream handles an update in the etcd stream prefix
func (t *Transcoder) handleStream(ctx context.Context, update *client.WatchUpdate) {
	if update.KV == nil {
		return
	}

	parts := strings.Split(string(update.KV.Key), ":")
	if len(parts) < 2 {
		return
	}
	name := parts[1]

	if len(parts) == 2 {
		t.handleStreamUpdate(ctx, name, update)
	} else if len(parts) == 3 && parts[2] == "transcoder" {
		t.handleStreamTranscoder(ctx, name, update)
	}
}

// handleStreamUpdate handles an etcd stream update
func (t *Transcoder) handleStreamUpdate(ctx context.Context, key string, update *client.WatchUpdate) {
	switch update.Type {
	case client.UpdateTypePut:
		var str stream.Stream
		err := json.Unmarshal(update.KV.Value, &str)
		if err != nil {
			log.Error().Err(err).Msg("stream unmarshal")
			return
		}
		t.streams[key] = &str
		// see if we should assign ourselves after a new stream was added
		if !t.shouldClaim() {
			break
		}
		t.claimStream(ctx, &str)

	case client.UpdateTypeDelete:
		delete(t.streams, key)
	}
	log.Debug().Msgf("streams %v", t.streams)
}

// handleStreamTranscoder handles an etcd stream transcoder update
func (t *Transcoder) handleStreamTranscoder(ctx context.Context, key string, update *client.WatchUpdate) {
	switch update.Type {
	case client.UpdateTypePut:
		t.streamTranscoders[key] = string(update.KV.Value)
	case client.UpdateTypeDelete:
		delete(t.jobs, key)

		stream, found := t.streams[key]
		if !found {
			break
		}
		// check if we should assign ourselves when another transcoder leaves
		if !t.shouldClaim() {
			break
		}
		t.claimStream(ctx, stream)
	}
	log.Debug().Msgf("streamsTranscoders %v", t.streamTranscoders)
}

// shouldClaim computes whether we should claim a slot for a certain service
func (t *Transcoder) shouldClaim() bool {
	if t.capacity-len(t.jobs) <= 0 {
		log.Info().Msg("Full capacity reached")
		return false
	}

	transcoders := make([]*TranscoderStatus, 0, len(t.transcoders))
	for _, transcoder := range t.transcoders {
		transcoders = append(transcoders, transcoder)
	}
	sort.Sort(ByLoad(transcoders))

	// Claim stream if we are the top candidate
	if len(transcoders) < 1 {
		return false
	}
	if transcoders[0].Name != t.name {
		return false
	}
	return true
}

// claimStream claims a stream for the current transcoder
func (t *Transcoder) claimStream(ctx context.Context, s *stream.Stream) {
	key := fmt.Sprintf("stream:%s:transcoder", s.Slug)
	lease, err := t.api.PublishWithLease(ctx, key, t.name, transcoderTTL)
	if err != nil {
		log.Error().Err(err).Msgf("transcoder/claim: %s", s.Slug)
		return
	}
	log.Info().Msgf("transcoder: claimed %s", s.Slug)
	// job, err := newJob(ctx, &jobConfig{
	// 	stream:     s,
	// 	lease:      lease,
	// 	api:        t.api,
	// 	configPath: t.configPath,
	// 	sinks:      []string{t.sink},
	// })
	job, err := t.createService(ctx, s, lease)
	if err != nil {
		log.Error().Err(err).Msgf("transcoder/claim: job %s", s.Slug)
	}
	t.jobs[s.Slug] = job
	t.publishStatus(ctx)
}

var configTemplate = template.Must(template.New("transcoderConfig").Parse(`
stream_key={{ .Slug }}
format={{ .Format }}
output={{ .OutputType }}
transcoding_source={{ .Source }}
transcoding_sink={{ index .Sinks 0 }}
`))

func (t *Transcoder) createService(ctx context.Context, s *stream.Stream, lease client.LeaseID) (*systemd.Service, error) {
	type StreamConfig struct {
		Slug       string
		Format     string
		OutputType string
		Source     string
		Sinks      []string
	}
	var buf bytes.Buffer
	err := configTemplate.Execute(&buf, &StreamConfig{
		Slug:   s.Slug,
		Format: s.Format,
		Source: s.Source,
		Sinks:  []string{t.sink},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("transcoder: templateConfig")
	}

	return systemd.NewService(ctx, &systemd.ServiceConfig{
		Config:     buf.Bytes(),
		ConfigPath: path.Join(t.configPath, s.Slug),
		UnitName:   fmt.Sprintf("transcode@%s.target", s.Slug),

		Lease: lease,
		API:   t.api,
		TTL:   transcoderTTL,
	})
}
