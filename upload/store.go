package upload

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Parser interface {
	ParsePlaylist(path string, playlist io.Reader) error
	Cleanup()
}

type ParserType string

const (
	ParserTypeHLS ParserType = "hls"
)

type Stream struct {
	ttl         time.Duration
	directories map[string]bool // directories

	origin        string
	originUpdated time.Time
	alive         chan struct{}

	mutex   sync.Mutex
	parsers map[ParserType]Parser
}

func (s *Stream) GetHLSParser(config HLSConfiguration) Parser {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.parsers[ParserTypeHLS] == nil {
		s.parsers[ParserTypeHLS] = newHLSParser(config)
	}
	return s.parsers[ParserTypeHLS]
}

type StreamStore struct {
	mutex    sync.Mutex
	done     sync.WaitGroup
	interval time.Duration
	data     map[string]*Stream // stream registration by slug
	log      zerolog.Logger
}

func NewStreamStore(ctx context.Context) *StreamStore {
	s := &StreamStore{
		interval: time.Second * 2,
		data:     make(map[string]*Stream),
		log:      log.With().Str("context", "store").Logger(),
	}

	s.done.Add(1)
	go func() {
		defer s.done.Done()
		ticker := time.NewTicker(s.interval)
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
	}()
	return s
}

// timeout removes all expired streams and files
func (s *StreamStore) timeout() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for slug, entry := range s.data {
		entry.ttl -= s.interval
		if entry.ttl <= 0 {
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
	close(stream.alive)
	for _, parser := range stream.parsers {
		parser.Cleanup()
	}
	delete(s.data, slug)
}

// Wait for store to stop
func (s *StreamStore) Wait() {
	s.done.Wait()
}

// Add new stream to store
// lock must be held by caller
func (s *StreamStore) addStream(slug string, origin string) {
	s.log.Info().Str("slug", slug).Msg("registering stream")
	s.data[slug] = &Stream{
		ttl:           time.Minute * 15, // initial timeout
		directories:   make(map[string]bool),
		origin:        origin,
		originUpdated: time.Now(),
		alive:         make(chan struct{}),
		parsers:       make(map[ParserType]Parser),
	}
}

var errInvalidOrigin = errors.New("invalid origin - this stream is already being uploaded from a different location")

// Check request origin and register stream if required
func (s *StreamStore) CheckOrigin(slug string, path string, origin string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	reg, ok := s.data[slug]
	if !ok {
		s.addStream(slug, origin)
		reg = s.data[slug]
	}

	now := time.Now()
	if origin != reg.origin {
		// disallow origin change before time out
		log.Debug().Msgf("%s %s %v", origin, reg.origin, reg.originUpdated.Add(reg.ttl).After(now))
		if reg.originUpdated.Add(reg.ttl).After(now) {
			return errInvalidOrigin
		}
	}

	reg.origin = origin
	reg.originUpdated = now
	reg.ttl = time.Minute * 15
	return nil
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
