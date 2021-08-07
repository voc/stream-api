package auth

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/voc/stream-api/client"
	"github.com/voc/stream-api/stream"
)

type watcher struct {
	api  client.WatchAPI
	done sync.WaitGroup

	// local state
	mutex    sync.Mutex
	settings map[string]*stream.Settings
}

func newWatcher(ctx context.Context, api client.ServiceAPI) *watcher {
	t := &watcher{
		api:      api,
		settings: make(map[string]*stream.Settings),
	}

	// watch source updates
	t.done.Add(1)
	go t.run(ctx)

	return t
}

func (w *watcher) Wait() {
	w.done.Wait()
}

// run keeps the communication to etcd
func (w *watcher) run(parentContext context.Context) {
	defer w.done.Done()
	ctx, cancel := context.WithCancel(parentContext)
	defer cancel()

	streamSettingsChan, err := w.api.Watch(ctx, client.StreamSettingsPrefix)
	if err != nil {
		log.Fatal().Err(err).Msg("auth: stream watch")
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case updates, ok := <-streamSettingsChan:
			if !ok {
				log.Fatal().Msg("auth: stream watch closed")
				return
			}
			for _, update := range updates {
				w.handleStreamSettings(ctx, update)
			}
		}
	}
}

// handleStream handles an update in the etcd stream prefix
func (w *watcher) handleStreamSettings(ctx context.Context, update *client.WatchUpdate) {
	if update.KV == nil {
		return
	}
	path := string(update.KV.Key)
	name := client.ParseStreamName(path)
	log.Debug().Msgf("stream settings update %v", update.KV.Value)
	if name == "" {
		return
	}

	w.mutex.Lock()
	defer w.mutex.Unlock()
	switch update.Type {
	case client.UpdateTypePut:
		var s stream.Settings
		err := json.Unmarshal(update.KV.Value, &s)
		if err != nil {
			log.Error().Err(err).Msg("auth: settings unmarshal")
			return
		}
		w.settings[name] = &s
	case client.UpdateTypeDelete:
		delete(w.settings, name)
	}
}

func (w *watcher) Auth(app string, slug string, secret string) bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	settings, ok := w.settings[slug]
	if !ok {
		return false
	}
	if app != settings.IngestType || secret != settings.Secret {
		return false
	}
	return true
}
