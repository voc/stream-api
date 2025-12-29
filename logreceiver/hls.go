package logreceiver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quangngotan95/go-m3u8/m3u8"
)

type HLSParser struct {
	streams map[string]*HLSStream
	client  *http.Client
	mutex   sync.RWMutex
}

type HLSStream struct {
	transport StreamTransport
	qualities map[string]*StreamId
	lastUsed  atomic.Value
	mutex     sync.Mutex
}

type HLSQuality struct {
	Type          StreamType
	EmbeddedAudio *StreamId
	Meta          string
}

func NewHLSParser(ctx context.Context, streamTimeout time.Duration) *HLSParser {
	if streamTimeout == 0 {
		streamTimeout = 15 * time.Minute
	}
	parser := &HLSParser{
		streams: make(map[string]*HLSStream),
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
	go parser.updatePlaylists(ctx, streamTimeout)
	return parser
}

func (h *HLSParser) updatePlaylists(ctx context.Context, streamTimeout time.Duration) {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.mutex.Lock()
			for slug, stream := range h.streams {
				if time.Since(stream.lastUsed.Load().(time.Time)) > streamTimeout {
					slog.Info("expiring stream", "slug", slug)
					delete(h.streams, slug)
					continue
				}
				go h.updateMasterPlaylist(slug, stream)
			}
			h.mutex.Unlock()
		}
	}
}

func (h *HLSParser) updateMasterPlaylist(slug string, stream *HLSStream) {
	slog.Debug("updating master playlist", "slug", slug)
	playlistUrl, err := h.masterPlaylistUrl(stream.transport, slug)
	if err != nil {
		slog.Warn("failed to get master playlist url", "slug", slug, "err", err)
		return
	}
	playlist, err := h.fetchPlaylist(playlistUrl)
	if err != nil {
		slog.Warn("failed to fetch playlist", "slug", slug, "path", playlistUrl, "err", err)
		return
	}

	// handle only master playlists
	if !playlist.IsMaster() {
		slog.Warn("fetched playlist is not master", "slug", slug, "path", playlistUrl)
		return
	}

	stream.mutex.Lock()
	defer stream.mutex.Unlock()
	stream.qualities = make(map[string]*StreamId)
	audioGroups := make(map[string]*StreamId)
	for _, item := range playlist.Items {
		switch val := item.(type) {
		case *m3u8.MediaItem:
			if val.Type != "AUDIO" {
				continue
			}
			quality := StreamId{
				Slug:      slug,
				Type:      StreamTypeAudio,
				Transport: stream.transport,
			}
			if val.Language != nil {
				quality.Meta = *val.Language
				audioGroups[val.GroupID] = &quality
			}
			if val.URI != nil {
				name := strings.Split(*val.URI, ".")[0]
				name = strings.TrimSuffix(name, "_stream") // mediamtx naming
				quality.Quality = StreamQuality(name)
				stream.qualities[name] = &quality
			}
		case *m3u8.PlaylistItem:
			name := strings.Split(val.URI, ".")[0]
			name = strings.TrimSuffix(name, "_stream") // mediamtx naming
			quality := StreamId{
				Slug:      slug,
				Type:      StreamTypeVideo,
				Transport: stream.transport,
				Quality:   StreamQuality(name),
			}
			if val.Height != nil && val.Width != nil {
				quality.Meta += fmt.Sprintf("%dx%d", *val.Width, *val.Height)
			}
			if val.Audio != nil {
				// audio, ok := audioGroups[*val.Audio]
				// if ok {
				// 	quality.EmbeddedAudio = audio
				// }
			}
			stream.qualities[name] = &quality
		}
	}
}

// voc-specific
func (h *HLSParser) masterPlaylistUrl(transport StreamTransport, slug string) (string, error) {
	switch transport {
	case StreamTransportHLS:
		return fmt.Sprintf("http://localhost/hls/%s/native_hd.m3u8", slug), nil
	case StreamTransportHLSLL:
		return fmt.Sprintf("http://localhost/hlsll/%s/index.m3u8", slug), nil
	}
	return "", fmt.Errorf("unknown transport: %s", transport)
}

// voc-specific
func (h *HLSParser) qualityFromSegmentURI(uri string, transport StreamTransport) (string, error) {
	switch transport {
	case StreamTransportHLS:
		return strings.Split(path.Base(uri), "_")[0], nil
	case StreamTransportHLSLL:
		parts := strings.Split(path.Base(uri), "_")
		if len(parts) < 2 {
			return "", fmt.Errorf("unexpected segment uri %q", uri)
		}
		return parts[1], nil
	}
	return "", fmt.Errorf("unknown transport: %q", transport)
}

func (h *HLSParser) fetchPlaylist(path string) (*m3u8.Playlist, error) {
	resp, err := http.Get(path)
	if err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}
	defer resp.Body.Close() // nolint

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %s code=%d", resp.Status, resp.StatusCode)
	}

	playlist, err := m3u8.Read(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	return playlist, nil
}

func (h *HLSParser) GetQuality(slug string, transport StreamTransport, file string) (*StreamId, bool) {
	// we only care about segment files
	if !strings.HasSuffix(file, ".ts") && !strings.HasSuffix(file, ".m4s") && !strings.HasSuffix(file, ".mp4") {
		// slog.Debug("not a segment file, wrong suffix", "file", file)
		return nil, false
	}
	h.mutex.RLock()
	stream, ok := h.streams[slug]
	h.mutex.RUnlock()
	if !ok {
		h.addStream(slug, transport)
		return nil, false
	}

	stream.mutex.Lock()
	defer stream.mutex.Unlock()
	stream.lastUsed.Store(time.Now())

	qualityName, err := h.qualityFromSegmentURI(file, transport)
	if err != nil {
		slog.Warn("failed to get quality from segment uri", "file", file, "err", err)
		return nil, false
	}
	quality, ok := stream.qualities[qualityName]
	if !ok {
		slog.Debug("quality not found in stream", "qualityName", qualityName, "quality", quality)
		return nil, false
	}

	return quality, true
}

func (h *HLSParser) addStream(slug string, transport StreamTransport) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	_, ok := h.streams[slug]
	if ok {
		return
	}
	stream := &HLSStream{
		qualities: make(map[string]*StreamId),
		transport: transport,
	}
	stream.lastUsed.Store(time.Now())
	h.streams[slug] = stream
	slog.Info("adding stream", "slug", slug)
}
