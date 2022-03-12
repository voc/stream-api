package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Sink struct {
	Address  string
	URL      url.URL
	Username string
	Password string
	queue    chan *http.Request
	done     sync.WaitGroup
}

func (sink *Sink) start(ctx context.Context, client *http.Client, numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		sink.done.Add(1)
		go sink.work(ctx, client)
	}
}

func (sink *Sink) handle(req *http.Request) {
	req.URL.Scheme = sink.URL.Scheme
	req.URL.Host = sink.URL.Host
	req.URL.Path, req.URL.RawPath = joinURLPath(&sink.URL, req.URL)
	req.SetBasicAuth(sink.Username, sink.Password)
	req.Response = nil
	req.RequestURI = ""
outer:
	for {
		select {
		case sink.queue <- req:
			break outer
		default:
			// drop front of queue
			<-sink.queue
			log.Info().Str("sink", sink.URL.Host).Msg("queue overflow")
			continue
		}
	}
}

func (sink *Sink) wait() {
	sink.done.Wait()
}

func (sink *Sink) work(ctx context.Context, client *http.Client) {
	defer sink.done.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-sink.queue:
		retry:
			for {
				select {
				case <-req.Context().Done():
					log.Warn().Str("sink", sink.URL.Host).Msg("discarding timed out request")
					break retry
				default:
				}
				res, err := client.Do(req)
				if err != nil {
					log.Error().Str("sink", sink.URL.Host).Err(err).Msg("sink error")
					break retry
				}
				res.Body.Close()
				if res.StatusCode != 200 {
					log.Warn().
						Str("sink", sink.URL.Host).
						Str("method", req.Method).
						Str("path", req.URL.Path).
						Str("status", res.Status).
						Msg("upload failed")
					// retry if we have space in queue
					time.Sleep(time.Second)
					continue
				}
				break
			}
		}
	}
}

type Proxy struct {
	sinks  []*Sink
	errors chan error
	client *http.Client
	ctx    context.Context
	done   sync.WaitGroup
}

func NewProxy(ctx context.Context, addr string, sinks []*Sink) *Proxy {
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConnsPerHost:   32,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	p := &Proxy{
		sinks:  sinks,
		client: &http.Client{Transport: tr},
		errors: make(chan error, 1),
		ctx:    ctx,
	}

	for _, sink := range p.sinks {
		sink.queue = make(chan *http.Request, 128)
	}

	mux := http.NewServeMux()
	srv := http.Server{Addr: addr, Handler: mux}

	// set routes
	mux.HandleFunc("/", p.HandleUpload)

	// run server
	p.done.Add(1)
	go func() {
		defer p.done.Done()
		err := srv.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			p.errors <- err
		}
	}()

	// close on ctx cancel
	p.done.Add(1)
	go func() {
		defer p.done.Done()
		<-ctx.Done()
		err := srv.Close()
		if err != nil {
			log.Error().Err(err).Msg("close")
		}
	}()

	// run sink uploaders
	for _, sink := range p.sinks {
		log.Printf("setup sink %v\n", sink)
		sink.start(ctx, p.client, 4)
	}

	return p
}

// Wait for server to finish
func (p *Proxy) Wait() {
	p.done.Wait()
	for _, sink := range p.sinks {
		sink.wait()
	}
}

// The channel returned by Errors is pushed fatal errors
func (p *Proxy) Errors() <-chan error {
	return p.errors
}

func getDeadline(path string) time.Time {
	ext := filepath.Ext(path)
	now := time.Now()
	switch ext {
	case ".m3u8":
		fallthrough
	case ".mpd":
		return now.Add(time.Second * 6)
	default:
		return now.Add(time.Second * 60)
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}

// do delete ourselves as delete doesnt work...
func (p *Proxy) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" && r.Method != "POST" && r.Method != "DELETE" {
		w.WriteHeader(200)
		log.Warn().Str("method", r.Method).Str("url", r.URL.Path).Msg("invalid method")
		io.WriteString(w, "Invalid method")
		return
	}

	deadline := getDeadline(r.URL.Path)
	ctx, _ := context.WithDeadline(p.ctx, deadline)
	log.Debug().Str("method", r.Method).Str("url", r.URL.Path).Msg("handle")

	var b bytes.Buffer
	b.ReadFrom(r.Body)
	getBody := func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewReader(b.Bytes())), nil
	}
	for _, sink := range p.sinks {
		req := r.Clone(ctx)
		req.ContentLength = r.ContentLength
		req.GetBody = getBody
		req.Body, _ = getBody()
		sink.handle(req)
	}
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}
