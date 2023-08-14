package upload

import (
	"errors"
	"io"
	"path/filepath"
	"time"
)

// Handler tracks the lifetime of uploaded streams and stream segments
type Handler struct {
	copier   FileCopier
	registry *FileRegistry
	store    *StreamStore

	playlistConfig PlaylistConfig

	maxPlaylistSize int
	maxSegmentSize  int
}

func NewHandler(config ServerConfig) *Handler {
	return &Handler{
		copier:   AtomicWriter{},
		registry: NewFileRegistry(FileRegistryConfig{}),
		store:    NewStreamStore(StreamStoreConfig{StreamTimeout: config.StreamTimeout, StreamOriginDuration: config.StreamOriginDuration}),

		playlistConfig: PlaylistConfig{
			Size: config.PlaylistSize,
		},
		maxPlaylistSize: config.MaxPlaylistSize,
		maxSegmentSize:  config.MaxSegmentSize,
	}
}

// Wait for registry to stop
func (h *Handler) Stop() {
	h.store.Stop()
	h.registry.Stop()
}

// Validates whether the request came from the usual origin and whether we could match a stream
// if we got a new master playlist
func (h *Handler) Validate(slug string, path string, origin string) error {
	return h.store.UpdateStream(slug, origin)
}

// handle file depending on extension
func (h *Handler) HandleFile(input io.Reader, slug string, outputPath string) error {
	stream := h.store.GetStream(slug)
	if stream == nil {
		return errors.New("unknown stream")
	}

	// keep track of directory
	// TODO: make sure we are not removing anything important
	dir := filepath.Dir(outputPath)
	h.registry.KeepFile(dir, stream.alive)

	var err error
	switch filepath.Ext(outputPath) {
	case ".m3u8":
		hls := stream.GetHLSParser(HLSConfiguration{
			slug:           slug,
			basePath:       dir,
			writer:         AtomicWriter{},
			registry:       h.registry,
			playlistConfig: h.playlistConfig,
		})
		src := LimitReads(input, int64(h.maxPlaylistSize))
		err = hls.ParsePlaylist(outputPath, src)
	case ".mpd":
		// err = s.HandlePlaylist(, file, , slug)

		// keep thumbnails/posters around
	case ".jpg":
		fallthrough
	case ".jpeg":
		h.registry.KeepFile(dir, stream.alive)
		fallthrough

	default:
		// write file and add basic expiry
		h.registry.RegisterFile(outputPath, time.Now().Add(time.Second*10))
		src := LimitReads(input, int64(h.maxSegmentSize))
		err = h.copier.CopyFile(outputPath, src)
	}

	return err
}
