package monitor

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/voc/stream-api/client"
	"github.com/voc/stream-api/fanout"
	"github.com/voc/stream-api/stream"
	"github.com/voc/stream-api/transcode"
)

type streamsJson struct {
	Streams map[string]stream.Stream `json:"streams"`
}

type transcodersJson struct {
	Transcoders map[string]transcode.TranscoderStatus `json:"transcoders"`
}

type streamTranscodersJson struct {
	StreamTranscoders map[string]string `json:"streamTranscoders"`
}

type fanoutsJson struct {
	Fanouts map[string]fanout.FanoutStatus `json:"fanouts"`
}

type watcher struct {
	api  client.WatchAPI
	done sync.WaitGroup

	// local state
	transcoders       map[string]*transcode.TranscoderStatus
	fanouts           map[string]*fanout.FanoutStatus
	streams           map[string]*stream.Stream
	streamTranscoders map[string]string

	updates chan map[string]interface{}
}

func newWatcher(ctx context.Context, api client.ServiceAPI) *watcher {
	t := &watcher{
		api:               api,
		fanouts:           make(map[string]*fanout.FanoutStatus),
		transcoders:       make(map[string]*transcode.TranscoderStatus),
		streams:           make(map[string]*stream.Stream),
		streamTranscoders: make(map[string]string),
		updates:           make(chan map[string]interface{}, 1),
	}

	// watch source updates
	t.done.Add(1)
	go t.run(ctx)

	return t
}

func (w *watcher) Wait() {
	w.done.Wait()
}

func (w *watcher) listen() <-chan map[string]interface{} {
	return w.updates
}

// run keeps the communication to etcd
func (w *watcher) run(parentContext context.Context) {
	defer w.done.Done()
	ctx, cancel := context.WithCancel(parentContext)
	defer cancel()

	transcoderChan, err := w.api.Watch(ctx, client.TranscoderPrefix)
	if err != nil {
		log.Fatal().Err(err).Msg("transcoder watch")
		return
	}
	fanoutChan, err := w.api.Watch(ctx, client.FanoutPrefix)
	if err != nil {
		log.Fatal().Err(err).Msg("fanout watch")
		return
	}
	streamChan, err := w.api.Watch(ctx, client.StreamPrefix)
	if err != nil {
		log.Fatal().Err(err).Msg("stream watch")
		return
	}
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
				w.handleTranscoder(update)
			}
		case updates, ok := <-fanoutChan:
			if !ok {
				log.Fatal().Msg("fanout watch closed")
				return
			}
			for _, update := range updates {
				w.handleFanout(update)
			}
		case updates, ok := <-streamChan:
			if !ok {
				log.Fatal().Msg("stream watch closed")
				return
			}
			for _, update := range updates {
				w.handleStream(ctx, update)
			}
		}
	}
}

// sendUpdate relays a state update
func (w *watcher) sendUpdate(key string, update interface{}) {
	tmp := make(map[string]interface{})
	tmp[key] = update
	w.updates <- tmp
}

// handleTranscoder handles an etcd transcoder update
func (w *watcher) handleTranscoder(update *client.WatchUpdate) {
	if update.KV == nil {
		return
	}
	name := client.ParseServiceName(string(update.KV.Key))
	if name == "" {
		return
	}

	switch update.Type {
	case client.UpdateTypePut:
		var status transcode.TranscoderStatus
		err := json.Unmarshal(update.KV.Value, &status)
		if err != nil {
			log.Error().Err(err).Msg("transcoder unmarshal")
			return
		}
		w.transcoders[name] = &status
	case client.UpdateTypeDelete:
		delete(w.transcoders, name)
	}
	log.Debug().Msgf("monitor/transcoders %v", w.transcoders)

	tmp := make(map[string]transcode.TranscoderStatus)
	for k, v := range w.transcoders {
		tmp[k] = *v
	}
	w.sendUpdate("transcoders", tmp)
}

// handleFanout handles an etcd fanout update
func (w *watcher) handleFanout(update *client.WatchUpdate) {
	if update.KV == nil {
		return
	}
	name := client.ParseServiceName(string(update.KV.Key))
	if name == "" {
		return
	}

	switch update.Type {
	case client.UpdateTypePut:
		var status fanout.FanoutStatus
		err := json.Unmarshal(update.KV.Value, &status)
		if err != nil {
			log.Error().Err(err).Msg("fanout unmarshal")
			return
		}
		w.fanouts[name] = &status
	case client.UpdateTypeDelete:
		delete(w.fanouts, name)
	}
	log.Debug().Msgf("monitor/fanouts %v", w.fanouts)

	tmp := make(map[string]fanout.FanoutStatus)
	for k, v := range w.fanouts {
		tmp[k] = *v
	}
	w.sendUpdate("fanouts", tmp)
}

// handleStream handles an update in the etcd stream prefix
func (w *watcher) handleStream(ctx context.Context, update *client.WatchUpdate) {
	if update.KV == nil {
		return
	}
	path := string(update.KV.Key)
	name := client.ParseStreamName(path)
	if name == "" {
		return
	}

	if client.PathIsStream(path) {
		w.handleStreamUpdate(ctx, name, update)
	} else if client.PathIsStreamTranscoder(path) {
		w.handleStreamTranscoder(ctx, name, update)
	}
}

// handleStreamUpdate handles an etcd stream update
func (w *watcher) handleStreamUpdate(ctx context.Context, key string, update *client.WatchUpdate) {
	switch update.Type {
	case client.UpdateTypePut:
		var str stream.Stream
		err := json.Unmarshal(update.KV.Value, &str)
		if err != nil {
			log.Error().Err(err).Msg("stream unmarshal")
			return
		}
		w.streams[key] = &str

	case client.UpdateTypeDelete:
		delete(w.streams, key)
	}
	log.Debug().Msgf("monitor/streams %v", w.streams)

	tmp := make(map[string]stream.Stream)
	for k, v := range w.streams {
		tmp[k] = *v
	}
	w.sendUpdate("streams", tmp)
}

// handleStreamTranscoder handles an etcd stream transcoder update
func (w *watcher) handleStreamTranscoder(ctx context.Context, key string, update *client.WatchUpdate) {
	switch update.Type {
	case client.UpdateTypePut:
		w.streamTranscoders[key] = string(update.KV.Value)
	case client.UpdateTypeDelete:
		delete(w.streamTranscoders, key)
	}
	log.Debug().Msgf("monitor/streamTranscoders %v", w.streamTranscoders)

	tmp := make(map[string]string)
	for k, v := range w.streamTranscoders {
		tmp[k] = v
	}
	w.sendUpdate("streamTranscoders", tmp)
}
