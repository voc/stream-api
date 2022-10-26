package upload

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type FileEntry struct {
	path     string          // path to the file
	deadline time.Time       // deadline for file deletion
	keep     <-chan struct{} // keep file as long as channel is active
}

// FileRegistry stores file paths+deadlines and removes the corresponding files when the deadline is reached
type FileRegistry struct {
	files map[string]*FileEntry
	add   chan *FileEntry
	done  sync.WaitGroup
}

func NewFileRegistry(ctx context.Context) *FileRegistry {
	r := &FileRegistry{
		files: make(map[string]*FileEntry),
		add:   make(chan *FileEntry, 1),
	}
	r.done.Add(1)
	go r.run(ctx)
	return r
}

func (r *FileRegistry) Wait() {
	r.done.Wait()
}

func (r *FileRegistry) run(ctx context.Context) {
	defer r.done.Done()
	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			r.cleanup()
			return
		case <-ticker.C:
			r.expire()
		case new := <-r.add:
			if entry, ok := r.files[new.path]; ok {
				if new.keep == nil && entry.keep != nil {
					continue
				}
			}
			r.files[new.path] = new
		}
	}
}

func (r *FileRegistry) cleanup() {
	for path := range r.files {
		log.Debug().Str("path", path).Msg("cleanup")
		err := os.RemoveAll(path)
		if err != nil {
			log.Error().Str("path", path).Err(err).Msg("remove failed")
		}
	}
}

func (r *FileRegistry) expire() {
	now := time.Now()
	for path, entry := range r.files {
		// prevents removing files with keep
		if entry.keep != nil {
			select {
			case <-entry.keep:
				// expire a few seconds from now
				entry.deadline = time.Now().Add(time.Second * 10)
				entry.keep = nil
				log.Debug().Msgf("keep expired %s", entry.path)
			default:
			}
			continue
		}

		if entry.deadline.After(now) {
			continue
		}

		log.Debug().Str("path", path).Msg("remove")
		err := os.RemoveAll(path)
		if err != nil {
			log.Error().Str("path", path).Err(err).Msg("remove failed")
		}
		delete(r.files, path)
	}
}

// Register a file path, the path will be removed after the deadline has expired
func (r *FileRegistry) RegisterFile(path string, deadline time.Time) {
	r.add <- &FileEntry{
		deadline: deadline,
		path:     path,
	}
}

// Keep file until keep channel is closed
func (r *FileRegistry) KeepFile(path string, keep <-chan struct{}) {
	r.add <- &FileEntry{
		path: path,
		keep: keep,
	}
}
