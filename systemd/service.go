package systemd

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

type CleanupFunc func()

// ServiceConfig represents config for a systemd service
type ServiceConfig struct {
	Config     []byte
	ConfigPath string
	UnitName   string
	Cleanup    CleanupFunc
}

// Service represents a single running systemd unit.
// The unit is kept alive until Stop is called.
type Service struct {
	conf *ServiceConfig
	conn *Conn

	done    sync.WaitGroup
	stopped atomic.Value
	cancel  context.CancelFunc
	ctx     context.Context
	mutex   sync.Mutex // mutex for preventing parallel unit and config access
}

// NewService creates a new Service
func NewService(parentContext context.Context, conf *ServiceConfig) (*Service, error) {
	conn, err := Connect()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(parentContext)
	s := &Service{
		conn:   conn,
		conf:   conf,
		cancel: cancel,
		ctx:    ctx,
	}
	s.stopped.Store(false)
	s.done.Add(1)
	go s.Run(ctx)
	return s, nil
}

// run starts the services maintenance loop
func (s *Service) Run(ctx context.Context) {
	defer s.done.Done()
	defer s.stopped.Store(true)
	defer s.cancel()

	s.start(ctx)
	defer s.stop()

	s.keepalive(ctx)

	<-ctx.Done()
}

func (s *Service) keepalive(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncUnitState(ctx)
		}
	}
}

// start deploys the config and restarts the unit if it changed
func (s *Service) start(ctx context.Context) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.deployConfig() {
		if err := s.conn.RestartUnit(ctx, s.conf.UnitName); err != nil {
			log.Error().Err(err).Msg("service: restartUnit")
		}
	} else {
		if err := s.conn.StartUnit(ctx, s.conf.UnitName); err != nil {
			log.Error().Err(err).Msg("service: startUnit")
		}
	}
	if err := s.conn.EnableUnit(ctx, s.conf.UnitName); err != nil {
		log.Error().Err(err).Msg("service: enableUnit")
	}
}

// stop disables/stops the unit and removes the config file
func (s *Service) stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if err := s.conn.DisableUnit(context.Background(), s.conf.UnitName); err != nil {
		log.Error().Err(err).Msg("service: disableUnit")
	}
	if err := s.conn.StopUnit(context.Background(), s.conf.UnitName); err != nil {
		log.Error().Err(err).Msg("service: stopUnit")
	}
	s.removeConfig()
	if s.conf.Cleanup != nil {
		s.conf.Cleanup()
	}
}

// deployConfig templates the service config
// if the file on disk changed it returns true
func (s *Service) deployConfig() bool {
	newConf := s.conf.Config

	// read old file for comparison
	oldConf, err := ioutil.ReadFile(s.conf.ConfigPath)
	if err == nil && bytes.Equal(oldConf, newConf) {
		return false
	}

	// write new file
	err = os.WriteFile(s.conf.ConfigPath, newConf, 0644)
	if err != nil {
		log.Error().Err(err).Msg("service: writeConfig")
	}
	return true
}

// removeConfig cleans up the config file
func (s *Service) removeConfig() {
	if err := os.Remove(s.conf.ConfigPath); err != nil {
		log.Error().Err(err).Msg("service: removeConfig")
	}
}

// syncState syncs transcoding jobs with running systemd units
// not needed if unit can't fail (Restart=always, StartLimitInterval=0)
func (s *Service) syncUnitState(ctx context.Context) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	units, err := s.conn.ListUnits(ctx)
	if err != nil {
		log.Error().Err(err).Msg("service: listUnits")
	}

	// reenable failed unit
	for _, unit := range units {
		if unit.Name != s.conf.UnitName {
			continue
		}

		if unit.ActiveState == "failed" {
			log.Info().Msg("service: restarting " + s.conf.UnitName)
			if err := s.conn.RestartUnit(ctx, s.conf.UnitName); err != nil {
				log.Error().Err(err).Msg("service: restartUnit")
			}
		}
	}
}

// Stop stops the service.
func (s *Service) Stop() {
	s.cancel()
}

// Wait waits for all of the services goroutines to finish.
// Must be called after Stop.
func (s *Service) Wait() {
	s.done.Wait()
}

// Stopped reports whether the service has stopped.
func (s *Service) Stopped() bool {
	return s.stopped.Load().(bool)
}

func (s *Service) Stopping() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}

// Restart service with new config
func (s *Service) Restart(newConfig []byte) {
	if s.Stopping() {
		return
	}
	s.mutex.Lock()
	s.conf.Config = newConfig
	s.mutex.Unlock()
	s.start(s.ctx)
}
