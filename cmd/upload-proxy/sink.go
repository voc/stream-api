package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/icholy/digest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	DefaultQueueSize = 128
	GracePeriod      = time.Minute * 5
)

type Sink struct {
	log       zerolog.Logger
	conf      SinkConfig
	url       url.URL
	transport *http.Transport
	reg       prometheus.Registerer
	queue     chan SinkEntry
	done      sync.WaitGroup

	graceDeadline time.Time
	deadlineMutex sync.Mutex

	cancel  context.CancelFunc
	metrics *SinkMetrics
}

type SinkEntry struct {
	Request  *http.Request
	QueuedAt time.Time
	Deadline time.Time
}

func NewSink(ctx context.Context, conf SinkConfig, transport *http.Transport, reg prometheus.Registerer) (*Sink, error) {
	url, err := url.Parse(conf.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid sink address: %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	s := &Sink{
		log:       log.With().Str("sink", conf.Address).Logger(),
		conf:      conf,
		url:       *url,
		reg:       reg,
		transport: transport,
		queue:     make(chan SinkEntry, conf.QueueSize),
		cancel:    cancel,
	}
	s.Start(ctx)
	return s, nil
}

func (sink *Sink) Address() string {
	return sink.conf.Address
}

func (sink *Sink) UpdateConfig(ctx context.Context, conf SinkConfig) error {
	sink.deadlineMutex.Lock()
	defer sink.deadlineMutex.Unlock()
	sink.graceDeadline = time.Time{} // reset grace deadline on config update
	if conf == sink.conf {
		// noop
		return nil
	}
	if conf.Address != sink.conf.Address {
		return fmt.Errorf("cannot update sink address")
	}
	if conf.QueueSize != sink.conf.QueueSize {
		return fmt.Errorf("cannot update sink queue size")
	}
	sink.Stop()
	sink.conf = conf
	sink.log.Info().Msg("sink config updated")
	ctx, cancel := context.WithCancel(ctx)
	sink.cancel = cancel
	sink.Start(ctx)
	return nil
}

func (sink *Sink) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	sink.cancel = cancel
	sink.metrics = NewSinkMetrics(sink.reg, sink.conf.Address)

	cli := &http.Client{Transport: sink.transport}
	if sink.conf.AuthType == AuthTypeDigest {
		cli = &http.Client{
			Transport: &digest.Transport{
				Username:  sink.conf.Username,
				Password:  sink.conf.Password,
				Transport: sink.transport,
			},
		}
	}

	sink.done.Add(1)
	go sink.work(ctx, cli)
	sink.log.Info().Msg("sink started")
}

func (sink *Sink) Stop() {
	sink.cancel()
	sink.done.Wait()
	sink.metrics.deregister()
	sink.log.Info().Msg("sink stopped")
}

func (sink *Sink) StartGracePeriod() {
	sink.deadlineMutex.Lock()
	defer sink.deadlineMutex.Unlock()
	sink.log.Warn().Msg("sink removed from config, starting grace period")
	if sink.graceDeadline.IsZero() {
		sink.graceDeadline = time.Now().Add(GracePeriod)
	}
}

func (sink *Sink) IsStale() bool {
	sink.deadlineMutex.Lock()
	defer sink.deadlineMutex.Unlock()
	return !sink.graceDeadline.IsZero() && time.Now().After(sink.graceDeadline)
}

func (sink *Sink) handle(req *http.Request, deadline time.Time) {
	req.URL.Scheme = sink.url.Scheme
	req.URL.Host = sink.url.Host
	req.URL.Path, req.URL.RawPath = joinURLPath(&sink.url, req.URL)
	if sink.conf.AuthType == AuthTypeBasic {
		req.SetBasicAuth(sink.conf.Username, sink.conf.Password)
	}
	req.Response = nil
	req.RequestURI = ""
outer:
	for {
		select {
		case sink.queue <- SinkEntry{Request: req, Deadline: deadline, QueuedAt: time.Now()}:
			break outer
		default:
			// drop front of queue
			<-sink.queue
			sink.metrics.totalNumDropped.Inc()
			log.Warn().Str("sink", sink.url.Host).Msg("queue full, dropping request")
			continue
		}
	}
}

func (sink *Sink) work(ctx context.Context, client *http.Client) {
	defer sink.done.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case entry := <-sink.queue:
			sink.upload(ctx, entry, client)
		}
	}
}

func (sink *Sink) upload(ctx context.Context, entry SinkEntry, client *http.Client) {
	for {
		if entry.Deadline.Before(time.Now()) {
			log.Warn().Str("sink", sink.url.Host).Msg("discarding timed out request")
			sink.metrics.totalNumDropped.Inc()
			return
		}
		queuedFor := time.Since(entry.QueuedAt)
		uploadStart := time.Now()
		res, err := client.Do(entry.Request)
		if err != nil {
			log.Error().Str("sink", sink.url.Host).Err(err).Msg("sink error")
			sink.metrics.totalNumDropped.Inc()
			return
		}
		_ = res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode > 299 {
			log.Warn().
				Str("sink", sink.url.Host).
				Str("method", entry.Request.Method).
				Str("path", entry.Request.URL.Path).
				Str("status", res.Status).
				Msg("upload failed")
			sink.metrics.totalTransientErrors.Inc()

			// get new body
			entry.Request.Body, _ = entry.Request.GetBody()

			// drop if queue is almost full
			if len(sink.queue) > cap(sink.queue)-2 {
				log.Error().Str("sink", sink.url.Host).Err(err).Msg("queue full, dropping request")
				sink.metrics.totalNumDropped.Inc()
				return
			}
			// retry after short delay
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}
		// success
		sink.metrics.totalNumUploaded.Inc()
		sink.metrics.totalQueueDelay.Add(queuedFor.Seconds())
		sink.metrics.totalUploadDuration.Add(time.Since(uploadStart).Seconds())
		sink.metrics.totalBytesUploaded.Add(float64(entry.Request.ContentLength))
		return
	}
}

type SinkMetrics struct {
	reg                  prometheus.Registerer
	totalNumUploaded     prometheus.Counter
	totalNumDropped      prometheus.Counter
	totalQueueDelay      prometheus.Counter
	totalUploadDuration  prometheus.Counter
	totalBytesUploaded   prometheus.Counter
	totalTransientErrors prometheus.Counter
}

func NewSinkMetrics(reg prometheus.Registerer, address string) *SinkMetrics {
	return &SinkMetrics{
		reg: reg,
		totalNumUploaded: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "upload_proxy_sink_uploaded_total",
			Help: "Number of uploaded requests",
			ConstLabels: prometheus.Labels{
				"sink": address,
			},
		}),
		totalNumDropped: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "upload_proxy_sink_dropped_total",
			Help: "Number of dropped requests due to error, timeout or full queue",
			ConstLabels: prometheus.Labels{
				"sink": address,
			},
		}),
		totalQueueDelay: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "upload_proxy_sink_queue_delay_total",
			Help: "Queue delay in seconds",
			ConstLabels: prometheus.Labels{
				"sink": address,
			},
		}),
		totalUploadDuration: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "upload_proxy_sink_upload_duration_total",
			Help: "Upload duration in seconds",
			ConstLabels: prometheus.Labels{
				"sink": address,
			},
		}),
		totalBytesUploaded: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "upload_proxy_sink_bytes_uploaded_total",
			Help: "Number of bytes uploaded",
			ConstLabels: prometheus.Labels{
				"sink": address,
			},
		}),
		totalTransientErrors: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "upload_proxy_sink_transient_errors_total",
			Help: "Number of transient errors",
			ConstLabels: prometheus.Labels{
				"sink": address,
			},
		}),
	}
}

func (m *SinkMetrics) deregister() {
	m.reg.Unregister(m.totalNumUploaded)
	m.reg.Unregister(m.totalNumDropped)
	m.reg.Unregister(m.totalQueueDelay)
	m.reg.Unregister(m.totalUploadDuration)
	m.reg.Unregister(m.totalBytesUploaded)
	m.reg.Unregister(m.totalTransientErrors)
}
