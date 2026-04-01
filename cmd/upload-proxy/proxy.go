package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

const MaxFileSize = 50 * 1024 * 1024 // 50 MB

type Proxy struct {
	sinks     []*Sink
	sinkMutex sync.Mutex
	errors    chan error
	transport *http.Transport
	ctx       context.Context
	done      sync.WaitGroup
	addr      string
	metrics   *ProxyMetrics
}

func NewProxy(ctx context.Context, conf Config) (*Proxy, error) {
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConnsPerHost:   32,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	p := &Proxy{
		transport: tr,
		errors:    make(chan error, 1),
		ctx:       ctx,
		metrics:   NewProxyMetrics(reg),
	}

	mux := http.NewServeMux()
	srv := http.Server{Addr: conf.ListenAddress, Handler: mux}

	// set routes
	mux.HandleFunc("/", p.HandleUpload)
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	ln, err := net.Listen("tcp", conf.ListenAddress)
	if err != nil {
		return nil, err
	}
	p.addr = ln.Addr().String()

	// run server
	p.done.Add(1)
	go func() {
		defer p.done.Done()
		err := srv.Serve(ln)
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

	p.done.Add(1)
	go p.runTimeout(ctx)

	// initial config update
	if err := p.UpdateConfig(ctx, conf); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Proxy) UpdateConfig(ctx context.Context, conf Config) error {
	p.sinkMutex.Lock()
	defer p.sinkMutex.Unlock()
	var newConfigs []SinkConfig
	var err error
	// mark removed sinks for graceful deletion
	for _, s := range p.sinks {
		var found bool
		for _, sinkConfig := range conf.Sinks {
			if s.Address() == sinkConfig.Address {
				found = true
				break
			}
		}
		if found {
			continue
		}
		s.StartGracePeriod()
	}
	// update existing sinks
	for _, sinkConfig := range conf.Sinks {
		var updated bool
		for _, s := range p.sinks {
			if s.Address() != sinkConfig.Address {
				continue
			}
			// update existing sink
			if err2 := s.UpdateConfig(ctx, sinkConfig); err2 != nil {
				log.Error().Err(err2).Str("sink", s.url.Host).Msg("sink update failed")
				firstErr(err2, &err)
			}
			updated = true
			break
		}
		if updated {
			continue
		}
		newConfigs = append(newConfigs, sinkConfig)
	}
	// create new sinks
	for _, sinkConfig := range newConfigs {
		sink, err2 := NewSink(ctx, sinkConfig, p.transport, p.metrics.reg)
		if err2 != nil {
			firstErr(err2, &err)
			log.Error().Err(err2).Str("sink", sinkConfig.Address).Msg("sink init failed")
			continue
		}
		p.sinks = append(p.sinks, sink)
		log.Info().Str("sink", sink.url.Host).Str("basePath", sink.url.Path).Str("authType", string(sink.conf.AuthType)).Str("username", sink.conf.Username).Msg("added sink")
	}
	return err
}

func (p *Proxy) runTimeout(ctx context.Context) {
	defer p.done.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.sinkMutex.Lock()
			sinks := p.sinks[:0]
			for _, s := range p.sinks {
				if s.IsStale() {
					s.Stop()
					log.Info().Str("sink", s.url.Host).Msg("removed sink after timeout")
					continue
				}
				sinks = append(sinks, s)
			}
			p.sinks = sinks
			p.sinkMutex.Unlock()
		}
	}
}

// Wait for server to finish
func (p *Proxy) Wait() {
	p.done.Wait()
	for _, sink := range p.sinks {
		sink.Stop()
	}
	p.metrics.deregister()
}

// The channel returned by Errors is pushed fatal errors
func (p *Proxy) Errors() <-chan error {
	return p.errors
}

func getDeadline(path string) time.Time {
	ext := filepath.Ext(path)
	now := time.Now()
	switch ext {
	case ".m3u8", ".mpd":
		return now.Add(time.Second * 6)
	case ".jpg", ".jpeg":
		return now.Add(time.Second * 20)
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

func (p *Proxy) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" && r.Method != "POST" && r.Method != "DELETE" {
		w.WriteHeader(200)
		log.Warn().Str("method", r.Method).Str("url", r.URL.Path).Msg("invalid method")
		_, _ = io.WriteString(w, "Invalid method")
		return
	}
	log.Debug().Str("method", r.Method).Str("url", r.URL.Path).Msg("handle")
	uploadStart := time.Now()
	var b bytes.Buffer
	size, err := b.ReadFrom(http.MaxBytesReader(w, r.Body, MaxFileSize))
	if err != nil {
		log.Error().Err(err).Msg("read body")
		w.WriteHeader(500)
		return
	}
	p.metrics.totalReadDelay.Add(time.Since(uploadStart).Seconds())
	p.metrics.totalNumRead.Inc()

	getBody := func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(b.Bytes())), nil
	}
	deadline := getDeadline(r.URL.Path)
	ctx, _ := context.WithDeadline(p.ctx, deadline)
	for _, sink := range p.sinks {
		req := r.Clone(ctx)
		req.ContentLength = size
		req.GetBody = getBody
		req.Body, _ = getBody()
		sink.handle(req, deadline)
	}
	w.WriteHeader(200)
}

func (p *Proxy) Address() string {
	return p.addr
}

func (p *Proxy) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("upload_proxy_sinks", "Number of configured sinks", nil, nil)
}

func (p *Proxy) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("upload_proxy_sinks", "Number of configured sinks", nil, nil),
		prometheus.GaugeValue,
		float64(len(p.sinks)),
	)
}

type ProxyMetrics struct {
	reg            prometheus.Registerer
	totalNumRead   prometheus.Counter
	totalReadDelay prometheus.Counter
}

func NewProxyMetrics(reg prometheus.Registerer) *ProxyMetrics {
	m := &ProxyMetrics{
		reg: reg,
		totalNumRead: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "upload_proxy_read_total",
			Help: "Total number of uploaded requests",
		}),
		totalReadDelay: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "upload_proxy_read_delay_seconds_total",
			Help: "Total upload delay in seconds",
		}),
	}
	return m
}

func (m *ProxyMetrics) deregister() {
	m.reg.Unregister(m.totalNumRead)
	m.reg.Unregister(m.totalReadDelay)
}

func firstErr(err error, errPtr *error) {
	if *errPtr == nil && err != nil {
		*errPtr = err
	}
}
