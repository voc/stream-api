package upload

import (
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

var errInvalidOrigin = errors.New("invalid origin - this stream is already being uploaded from a different location")

type Stream struct {
	timeout     time.Duration
	ttl         time.Duration
	directories map[string]bool // directories

	origin        string
	originUpdated time.Time

	// indicates whether this stream is still alive, may be used to
	// trigger cleanup of associated data
	alive chan struct{}

	parserMutex sync.Mutex
	parsers     map[ParserType]Parser
}

func NewStream(timeout time.Duration) *Stream {
	return &Stream{
		timeout:     timeout,
		ttl:         timeout, // initial timeout
		directories: make(map[string]bool),
		alive:       make(chan struct{}),
		parsers:     make(map[ParserType]Parser),
	}
}

// update stream origin and ttl
func (s *Stream) Update(origin string) error {
	now := time.Now()
	if origin != s.origin {
		// disallow origin change before 1/2 timeout
		log.Debug().Msgf("%s %s %v", origin, s.origin, s.originUpdated.Add(s.timeout/2).After(now))
		if s.originUpdated.Add(s.timeout / 2).After(now) {
			return errInvalidOrigin
		}
	}

	s.origin = origin
	s.originUpdated = now
	s.ttl = s.timeout
	return nil
}

// returns true if the stream should be expired
func (s *Stream) Age(duration time.Duration) bool {
	s.ttl -= duration
	return s.ttl <= 0
}

func (s *Stream) Cleanup() {
	s.parserMutex.Lock()
	defer s.parserMutex.Unlock()
	close(s.alive)
	for k, parser := range s.parsers {
		parser.Cleanup()
		delete(s.parsers, k)
	}
}

func (s *Stream) GetHLSParser(config HLSConfiguration) Parser {
	s.parserMutex.Lock()
	defer s.parserMutex.Unlock()
	if s.parsers[ParserTypeHLS] == nil {
		s.parsers[ParserTypeHLS] = newHLSParser(config)
	}
	return s.parsers[ParserTypeHLS]
}
