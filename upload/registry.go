package upload

import (
	"context"
	"log"
	"sync"
	"time"
)

type registration struct {
	ttl     time.Duration
	timeout time.Duration
	files   map[string]time.Duration // uploaded files with expirations
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

type CleanupFunc func(slug string, path *string)

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
					r.callback(slug, nil)
					delete(r.data, slug)
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
			r.callback(slug, nil)
			delete(r.data, slug)
			continue
		}
		for path, lifetime := range entry.files {
			lifetime -= r.interval
			// log.Println("check", slug, path, lifetime)
			if lifetime <= 0 {
				r.callback(slug, &path)
				delete(entry.files, path)
				continue
			}
			// update lifetime
			entry.files[path] = lifetime
		}
	}
}

func (r *Registry) Wait() {
	r.done.Wait()
}

// add new stream to registry
// lock must be held by caller
func (r *Registry) addStream(slug string) {
	log.Println("registering stream", slug)
	r.data[slug] = &registration{
		ttl:     time.Second * 10, // initial timeout
		timeout: time.Second * 10,
		files:   make(map[string]time.Duration),
	}
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
	reg.files[path] = reg.timeout
	log.Println("add file ", path, reg.timeout)
	return nil
}

// Keepalive extends the lifetime of a stream and its files
func (r *Registry) Keepalive(slug string, paths []string, timeout time.Duration) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	_, ok := r.data[slug]
	if !ok {
		r.addStream(slug)
	}

	// Refresh lifetimes
	reg := r.data[slug]
	reg.ttl = timeout
	reg.timeout = timeout
	for _, path := range paths {
		reg.files[path] = timeout
	}
}
