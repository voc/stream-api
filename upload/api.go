package upload

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/rs/zerolog/log"
)

type APIServer struct {
	srv *http.Server
}

func NewAPIServer(address string) *APIServer {
	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Error().Err(err).Msg("failed to listen")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/streams", handleStreams())
	srv := &http.Server{
		Handler: mux,
	}

	go func() {
		err := srv.Serve(l)
		if err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("failed to serve")
		}
	}()

	return &APIServer{
		srv: srv,
	}
}

func (s *APIServer) Stop() {
	err := s.srv.Close()
	if err != nil {
		log.Error().Err(err).Msg("failed to close server")
	}
}

type HLSStream struct{}

type StreamDetails interface{}

type StreamDescription struct {
	// stream type
	Type    string          `json:"type"`
	Details json.RawMessage `json:"details"`
}

type StreamResponse struct {
	Slug string `json:"slug"`

	// optional thumbnail url
	ThumbnailURL string `json:"thumbnail,omitempty"`

	// optional poster url
	PosterURL string `json:"poster,omitempty"`

	Streams  []StreamDescription `json:"streams,omitempty"`
	Playlist string              `json:"playlist,omitempty"`
}

func handleStreams() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// json.Marshal(v)
	}
}
