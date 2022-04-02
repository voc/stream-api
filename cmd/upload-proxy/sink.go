package main

import (
	"context"
	"net/http"
	"net/url"
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
