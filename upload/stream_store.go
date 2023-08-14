package upload

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	DefaultStreamTimeout        = time.Second * 10
	DefaultStreamExpireInterval = time.Second * 2
	DefaultStreamOriginDuration = time.Second * 6
)

type StreamStoreConfig struct {
	// how long a stream can be idle (no input) before being removed
	StreamTimeout time.Duration

	// how often to expire old streams
	StreamExpireInterval time.Duration

	// for how long to keep the stream origin, after no more data is received
	StreamOriginDuration time.Duration
}

type StreamStore struct {
	config StreamStoreConfig
	data   map[string]*Stream // stream registration by slug
	log    zerolog.Logger

	mutex  sync.Mutex
	cancel context.CancelFunc
	done   sync.WaitGroup
}

func NewStreamStore(config StreamStoreConfig) *StreamStore {
	if config.StreamTimeout == 0 {
		config.StreamTimeout = DefaultStreamTimeout
	}
	if config.StreamExpireInterval == 0 {
		config.StreamExpireInterval = DefaultStreamExpireInterval
	}
	if config.StreamOriginDuration == 0 {
		config.StreamOriginDuration = DefaultStreamOriginDuration
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &StreamStore{
		config: config,
		data:   make(map[string]*Stream),
		log:    log.With().Str("context", "store").Logger(),
		cancel: cancel,
	}

	s.done.Add(1)
	go s.run(ctx)
	return s
}

// run expire routine
func (s *StreamStore) run(ctx context.Context) {
	defer s.done.Done()
	ticker := time.NewTicker(s.config.StreamExpireInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// cleanup all
			s.mutex.Lock()
			for slug := range s.data {
				s.removeStream(slug)
			}
			s.mutex.Unlock()
			return
		case <-ticker.C:
			// count down timeout
			s.timeout()
		}
	}
}

// timeout removes all expired streams and files
func (s *StreamStore) timeout() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for slug, stream := range s.data {
		if stream.Age(s.config.StreamExpireInterval) {
			s.removeStream(slug)
		}
	}
}

// remove tracked stream
// lock must be held by caller
func (s *StreamStore) removeStream(slug string) {
	s.log.Info().Str("slug", slug).Msg("removing stream")
	stream, ok := s.data[slug]
	if !ok {
		return
	}
	stream.Cleanup()
	delete(s.data, slug)
}

// Wait for store to stop
func (s *StreamStore) Stop() {
	s.cancel()
	s.done.Wait()
}

// Check request origin and register stream if required
func (s *StreamStore) UpdateStream(slug string, origin string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	stream, ok := s.data[slug]
	if !ok {
		s.log.Info().Str("slug", slug).Msg("registering stream")
		stream = NewStream(s.config.StreamTimeout, s.config.StreamOriginDuration)
		s.data[slug] = stream
	}

	return stream.Update(origin)
}

// Get stream by slug
func (s *StreamStore) GetStream(slug string) *Stream {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	reg, ok := s.data[slug]
	if !ok {
		return nil
	}
	return reg
}
