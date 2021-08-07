package monitor

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/voc/stream-api/client"
	"github.com/voc/stream-api/config"
)

type Monitor struct {
	server *server
}

func New(ctx context.Context, conf config.MonitorConfig, api client.ServiceAPI) *Monitor {
	log.Debug().Msgf("monitor config %v", conf)
	watcher := newWatcher(ctx, api)
	m := &Monitor{
		server: newServer(ctx, api, watcher.listen(), conf),
	}

	return m
}

func (m *Monitor) Wait() {
	m.server.Wait()
}
