package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ProxyRequest struct {
	req      *http.Request
	deadline time.Time
}

type Sink struct {
	Address  string
	URL      url.URL
	Username string
	Password string
	queue    chan *http.Request
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
		MaxIdleConnsPerHost: 16,
		IdleConnTimeout:     60 * time.Second,
	}
	p := &Proxy{
		sinks:  sinks,
		client: &http.Client{Transport: tr},
		errors: make(chan error, 1),
		ctx:    ctx,
	}

	for _, sink := range p.sinks {
		sink.queue = make(chan *http.Request, 32)
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
			log.Println("close:", err)
		}
	}()

	// run sink uploaders
	for _, sink := range sinks {
		p.done.Add(1)
		log.Printf("setup sink %v\n", sink)
		go func(sink *Sink) {
			defer p.done.Done()
			// todo: implement upload workers for parallel upload
			// and avoiding blocking
			for {
				select {
				case <-ctx.Done():
					return
				case req := <-sink.queue:
					req.URL.Scheme = sink.URL.Scheme
					req.URL.Host = sink.URL.Host
					req.URL.Path, req.URL.RawPath = joinURLPath(&sink.URL, req.URL)
					req.SetBasicAuth(sink.Username, sink.Password)
					req.Response = nil
					req.RequestURI = ""

				retry:
					for {
						select {
						case <-req.Context().Done():
							log.Println("discarding timed out request")
							break retry
						default:
						}
						res, err := p.client.Do(req)
						// todo: retry, timeout
						if err != nil {
							log.Println("sink err", err)
							break retry
						}
						res.Body.Close()
						if res.StatusCode != 200 {
							log.Println("upload failed", req.URL.Path, sink.URL.Host, res.Status)
							// retry if we have space in queue
							time.Sleep(time.Second)
							continue
						}
						break
					}
				}
			}
		}(sink)
	}

	return p
}

// Wait for server to finish
func (p *Proxy) Wait() {
	p.done.Wait()
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
		return now.Add(time.Second * 8)
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
		log.Println("invalid method", r.Method, r.URL.Path)
		io.WriteString(w, "Invalid method")
		return
	}

	deadline := getDeadline(r.URL.Path)
	ctx, _ := context.WithDeadline(p.ctx, deadline)
	log.Println("handle", r.URL.Path)

	var b bytes.Buffer
	b.ReadFrom(r.Body)
	for _, sink := range p.sinks {
	outer:
		for {
			req := r.Clone(ctx)
			req.ContentLength = r.ContentLength
			req.Body = ioutil.NopCloser(bytes.NewReader(b.Bytes()))
			select {
			case sink.queue <- req:
				break outer
			default:
				// drop front of queue
				<-sink.queue
				log.Println("sink", sink.URL.Host, "queue overflow")
				continue
			}
		}
	}
	w.WriteHeader(200)
	w.Write([]byte("ok"))
	// r.Body.Close()
}
