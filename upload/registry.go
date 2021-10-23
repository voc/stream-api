package upload

import (
	"context"
	"log"
	"path/filepath"
	"sync"
	"time"
)

type registration struct {
	ttl         time.Duration
	directories map[string]bool // directories
}

// Registry tracks the lifetime of uploaded streams and stream segments
type Registry struct {
	ctx      context.Context
	data     map[string]*registration // stream registration by slug
	mutex    sync.Mutex
	done     sync.WaitGroup
	interval time.Duration
	callback CleanupFunc
}

type CleanupFunc func(slug string, path string)

func NewRegistry(ctx context.Context, callback CleanupFunc) *Registry {
	r := &Registry{
		ctx:      ctx,
		data:     make(map[string]*registration),
		interval: time.Second * 2,
		callback: callback,
	}

	r.done.Add(1)
	go func() {
		defer r.done.Done()
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				// cleanup all
				r.mutex.Lock()
				for slug := range r.data {
					r.removeStream(slug)
				}
				r.mutex.Unlock()
				return
			case <-ticker.C:
				// count down timeout
				r.timeout()
			}
		}
	}()
	return r
}

// timeout removes all expired streams and files
func (r *Registry) timeout() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for slug, entry := range r.data {
		entry.ttl -= r.interval
		if entry.ttl <= 0 {
			r.removeStream(slug)
		}

	}
}

// add new stream to registry
// lock must be held by caller
func (r *Registry) addStream(slug string) {
	log.Println("registering stream", slug)
	r.data[slug] = &registration{
		ttl:         time.Second * 10, // initial timeout
		directories: make(map[string]bool),
	}
}

// remove tracked stream
// lock must be held by caller
func (r *Registry) removeStream(slug string) {
	log.Println("removing stream", slug)
	entry, ok := r.data[slug]
	if !ok {
		return
	}
	for dir := range entry.directories {
		r.callback(slug, dir)
	}
	delete(r.data, slug)
}

// Wait for registry to stop
func (r *Registry) Wait() {
	r.done.Wait()
}

// Addfile adds a path to track for a certain stream
// if the stream doesn't exist yet we assume a timeout, the manifest will have to be uploaded shortly after
func (r *Registry) AddFile(slug string, path string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	reg, ok := r.data[slug]
	if !ok {
		r.addStream(slug)
		reg = r.data[slug]
	}
	dir := filepath.Dir(path)
	reg.directories[dir] = true
	return nil
}

// Keepalive extends the lifetime of a stream and its directories
func (r *Registry) Keepalive(slug string, timeout time.Duration) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	_, ok := r.data[slug]
	if !ok {
		r.addStream(slug)
	}

	// Refresh lifetime
	reg := r.data[slug]
	reg.ttl = timeout
}
