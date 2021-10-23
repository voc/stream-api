package upload

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func fail(w http.ResponseWriter, err error) {
	log.Println("fail", err)
	w.WriteHeader(500)
	io.WriteString(w, err.Error())
}

type Server struct {
	registry  *Registry
	auth      Auth
	storePath string
	parser    Parser
	errors    chan error
	done      sync.WaitGroup
}

func NewServer(ctx context.Context, addr string, path string, auth Auth) *Server {
	s := &Server{
		auth:      auth,
		storePath: path,
		errors:    make(chan error, 1),
	}
	s.registry = NewRegistry(ctx, s.cleanup)

	mux := http.NewServeMux()
	srv := http.Server{Addr: addr, Handler: mux}

	// set routes
	mux.HandleFunc("/", s.HandleUpload)

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
			log.Println("close:", err)
		}
	}()
	return s
}

// Wait for server to finish
func (s *Server) Wait() {
	s.registry.Wait()
	s.done.Wait()
}

// The channel returned by Errors is pushed fatal errors
func (s *Server) Errors() <-chan error {
	return s.errors
}

// do delete ourselves as delete doesnt work...
func (s *Server) HandleUpload(w http.ResponseWriter, r *http.Request) {
	// do auth
	// handle post/put
	// make sure to update files before playlist
	// disect path
	// register path-timeout
	// -> cleanup if path times out

	if !s.authenticate(w, r) {
		w.WriteHeader(401)
		io.WriteString(w, "Unauthorized")
		return
	}

	if r.Method == "PUT" || r.Method == "POST" {
		path := filepath.Join(s.storePath, r.URL.Path)
		dir := filepath.Dir(path)
		ext := filepath.Ext(path)
		tmpPath := path + ".tmp"
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			fail(w, fmt.Errorf("mkdir: %w", err))
		}
		// log.Println("allow", r.Method, r.URL.Path)

		file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
		if err != nil {
			fail(w, err)
			return
		}
		defer file.Close()

		switch ext {
		case ".m3u8":
			fallthrough
		case ".mpd":
			err = s.HandlePlaylist(r.Body, file, r.URL.Path)
		default:
			err = s.HandleSegment(r.Body, file, r.URL.Path)
		}
		if err != nil {
			fail(w, err)
			return
		}
		err = os.Rename(tmpPath, path)
		if err != nil {
			fail(w, fmt.Errorf("rename: %w", err))
		}
	} else if r.Method == "DELETE" {
		path := filepath.Join(s.storePath, r.URL.Path)
		os.Remove(path)
	} else {
		log.Println("unhandled", r.Method, r.URL.Path, r.Header.Get("Authorization"))
	}
}

func (s *Server) HandlePlaylist(body io.Reader, output io.Writer, path string) error {
	var buf bytes.Buffer
	writer := io.MultiWriter(&buf, output)
	_, err := io.Copy(writer, body)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	// parse playlist
	dir := filepath.Dir(path)
	slug := filepath.Base(dir)
	ext := filepath.Ext(path)
	// var paths []string
	var interval time.Duration
	switch ext {
	case ".m3u8":
		_, interval, err = s.parser.parseHLSPlaylist(&buf, dir)
	case ".mpd":
		_, interval, err = s.parser.parseDashManifest(&buf, dir)
	default:
		log.Fatal("unknown playlist extension", ext)
	}

	if err != nil {
		return fmt.Errorf("playlist parse: %w", err)
	}

	// refresh stream registration

	s.registry.Keepalive(slug, nil, interval*2)

	return nil
}

func (s *Server) HandleSegment(body io.Reader, output io.Writer, path string) error {
	_, err := io.Copy(output, body)

	// add path to registry
	// slug := filepath.Base(filepath.Dir(path))
	// s.registry.AddFile(slug, path)
	return err
}

// Authenticate using basic auth
func (s *Server) authenticate(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Add("WWW-Authenticate", `Basic realm=upload, charset="UTF-8"`)
	auth := strings.Trim(r.Header.Get("Authorization"), "\n")
	if len(auth) > 6 && strings.ToLower(auth[:6]) != "basic " {
		return false
	}
	split := strings.Split(auth, " ")
	if len(split) != 2 {
		return false
	}
	buf := bytes.NewBuffer([]byte(split[1]))
	rd := base64.NewDecoder(base64.StdEncoding, buf)
	res, err := io.ReadAll(rd)
	if err != nil {
		return false
	}
	split = strings.Split(strings.ReplaceAll(string(res), "\n", ""), ":")
	if len(split) != 2 {
		return false
	}
	ret := s.auth.Auth(split[0], split[1], r.URL.Path)
	// log.Println("split", split[0], split[1], ret)
	return ret
}

// cleanup removes a stream or single stream-segment
func (s *Server) cleanup(slug string, path *string) {
	// remove whole directory
	streamPath := filepath.Join(s.storePath, slug)
	if path == nil {
		log.Println("remove dir", streamPath)
		err := os.RemoveAll(streamPath)
		if err != nil {
			log.Println("remove stream:", err)
		}
		return
	}

	// just a sanity check
	// if !strings.HasPrefix(*path, "/"+slug+"/") {
	// 	log.Printf("attempted to remove %s not belonging to slug %s", *path, slug)
	// 	return
	// }

	// filePath := filepath.Join(s.storePath, *path)
	// log.Println("remove file", filePath)
	// err := os.Remove(filePath)
	// if err != nil {
	// 	log.Println("remove:", err)
	// }
}
