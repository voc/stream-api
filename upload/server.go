package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

func fail(w http.ResponseWriter, err error) {
	log.Error().Err(err).Msg("fail")
	w.WriteHeader(500)
	_, _ = io.WriteString(w, err.Error())
}

type ServerConfig struct {
	Addr            string
	OutputPath      string
	MaxSegmentSize  int
	MaxPlaylistSize int
	StreamTimeout   time.Duration

	PlaylistSize int
}

type Server struct {
	handler    *Handler
	auth       Auth
	outputPath string
	errors     chan error
	cancel     context.CancelFunc
	done       sync.WaitGroup
}

func NewServer(auth Auth, config ServerConfig) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		handler:    NewHandler(config),
		auth:       auth,
		outputPath: config.OutputPath,
		errors:     make(chan error, 1),
		cancel:     cancel,
	}
	mux := http.NewServeMux()
	srv := http.Server{Addr: config.Addr, Handler: mux}

	// set routes
	mux.HandleFunc("/", s.HandleUpload)
	mux.HandleFunc("/health", s.HandleHealth)

	// run server
	s.done.Add(1)
	go func() {
		defer s.done.Done()
		err := srv.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			s.errors <- err
		}
	}()

	// close on ctx cancel
	s.done.Add(1)
	go func() {
		defer s.done.Done()
		<-ctx.Done()
		err := srv.Close()
		if err != nil {
			log.Error().Err(err).Msg("close")
		}
	}()
	return s
}

// Wait for server to finish cleaning up
func (s *Server) Stop() {
	s.cancel()
	s.done.Wait()
	s.handler.Stop()
}

// The channel returned by Errors receives fatal errors
func (s *Server) Errors() <-chan error {
	return s.errors
}

func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	_, _ = io.WriteString(w, "ok")
}

func (s *Server) HandleUpload(w http.ResponseWriter, req *http.Request) {
	// do auth
	// handle post/put
	// make sure to update files before playlist
	// disect path
	// register path-timeout
	// -> cleanup if path times out

	slug, ok := s.authenticate(w, req)
	if !ok {
		w.WriteHeader(401)
		_, _ = io.WriteString(w, "Unauthorized")
		return
	}

	path := filepath.Join(s.outputPath, req.URL.Path)
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		fail(w, err)
		return
	}
	if err := s.handler.Validate(slug, path, host); err != nil {
		w.WriteHeader(403)
		msg := fmt.Sprintf("Request blocked: %s", err.Error())
		log.Debug().Msg(msg)
		_, _ = io.WriteString(w, msg)
		return
	}

	if req.Method == "PUT" || req.Method == "POST" {
		err := s.handler.HandleFile(req.Body, slug, path)
		if err != nil {
			fail(w, err)
			return
		}

	} else if req.Method == "DELETE" {
		// ignore delete requests
		// path := filepath.Join(s.storePath, req.URL.Path)
		// os.Remove(path)
	} else {
		log.Debug().Str("method", req.Method).Str("path", req.URL.Path).Msg("unhandled")
		w.WriteHeader(405)
		_, _ = io.WriteString(w, "Method Not Allowed")
	}
}

// Authenticate using basic auth
func (s *Server) authenticate(w http.ResponseWriter, r *http.Request) (string, bool) {
	w.Header().Add("WWW-Authenticate", `Basic realm=upload, charset="UTF-8"`)
	username, password, ok := r.BasicAuth()
	if !ok {
		return "", false
	}
	slug, ret := s.auth.Auth(username, password, r.URL.Path)
	return slug, ret
}
