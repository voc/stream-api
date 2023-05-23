package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const DefaultQueueSize = 128

type SinkConfig struct {
	Address   string
	Username  string
	Password  string
	QueueSize int `toml:"queue-size"`
}

type Sink struct {
	conf  SinkConfig
	url   url.URL
	queue chan *http.Request
	done  sync.WaitGroup
}

func NewSink(conf SinkConfig) (*Sink, error) {
	if conf.QueueSize == 0 {
		conf.QueueSize = DefaultQueueSize
	}

	url, err := url.Parse(conf.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid sink address: %w", err)
	}
	return &Sink{
		conf:  conf,
		url:   *url,
		queue: make(chan *http.Request, conf.QueueSize),
	}, nil
}

func (sink *Sink) start(ctx context.Context, client *http.Client, numWorkers int) {
	sink.done.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go sink.work(ctx, client)
	}
}

func (sink *Sink) handle(req *http.Request) {
	req.URL.Scheme = sink.url.Scheme
	req.URL.Host = sink.url.Host
	req.URL.Path, req.URL.RawPath = joinURLPath(&sink.url, req.URL)
	req.SetBasicAuth(sink.conf.Username, sink.conf.Password)
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
			log.Info().Str("sink", sink.url.Host).Msg("queue overflow")
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
					log.Warn().Str("sink", sink.url.Host).Msg("discarding timed out request")
					break retry
				default:
				}
				res, err := client.Do(req)
				if err != nil {
					log.Error().Str("sink", sink.url.Host).Err(err).Msg("sink error")
					break retry
				}
				res.Body.Close()
				if res.StatusCode != 200 {
					log.Warn().
						Str("sink", sink.url.Host).
						Str("method", req.Method).
						Str("path", req.URL.Path).
						Str("status", res.Status).
						Msg("upload failed")

					// get new body
					req.Body, _ = req.GetBody()

					// retry if we have space in queue
					time.Sleep(time.Second)
					continue
				}
				break
			}
		}
	}
}
