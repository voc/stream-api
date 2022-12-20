package upload

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/quangngotan95/go-m3u8/m3u8"
)

type HLSParser struct {
	mutex          sync.Mutex
	slug           string
	basePath       string
	playlistConfig PlaylistConfig
	writer         FileWriter
	registry       *FileRegistry
	subs           map[string]*VariantPlaylist // tracked playlists
	files          map[string]chan struct{}    // tracked files
}

type HLSConfiguration struct {
	slug           string
	basePath       string
	playlistConfig PlaylistConfig
	writer         FileWriter
	registry       *FileRegistry
}

type PlaylistConfig struct {
	Size int
}

// Tracks a certain source playlist and produces a continuous output playlist
type VariantPlaylist struct {
	lastIndex    int
	lastSize     int
	lastSequence int
	output       *LivePlaylist
}

func newHLSParser(config HLSConfiguration) *HLSParser {
	h := &HLSParser{
		slug:           config.slug,
		basePath:       config.basePath,
		playlistConfig: config.playlistConfig,
		writer:         config.writer,
		registry:       config.registry,
		subs:           make(map[string]*VariantPlaylist),
		files:          make(map[string]chan struct{}),
	}
	return h
}

func (h *HLSParser) Cleanup() {
	for _, keep := range h.files {
		close(keep)
	}
	h.files = nil
}

/*
Process hls playlist
 1. parse playlist
 2. if master playlist:
    - fix up codec information
    - output unchanged and generate language specific/sd master playlists
 3. if segment playlist:
    - insert discontinuity if sequence went backwards
    - append new segments to output playlist
*/
func (h *HLSParser) ParsePlaylist(path string, reader io.Reader) error {
	playlist, err := m3u8.Read(reader)
	if err != nil {
		return err
	}

	h.mutex.Lock()
	defer h.mutex.Unlock()

	// handle master playlist
	if playlist.IsMaster() {
		fixupCodecInformation(playlist)
		if err := h.writePlaylist(path, playlist); err != nil {
			return err
		}
		if err := h.generateLanguageMasters(path, "hd", playlist); err != nil {
			return err
		}
		if !filterHDResolutions(playlist) {
			// no non-hd resolutions, no need to generate sd master playlist
			return nil
		}
		if err := h.generateLanguageMasters(path, "sd", playlist); err != nil {
			return err
		}
		return nil
	}

	// lookup playlist by path
	v, err := h.getVariantPlaylist(path)
	if err != nil {
		return err
	}
	h.checkDiscontinuity(v, playlist)
	h.appendItems(v, playlist)
	v.output.setSegmentTarget(playlist.Target)

	// write to disk
	return h.writePlaylist(path, &v.output.Playlist)
}

// Beware special VOC hack:
// Generate separate master playlist for every non-default language variant
func (h *HLSParser) generateLanguageMasters(path string, suffix string, playlist *m3u8.Playlist) error {
	dir := filepath.Dir(path)
	lang := getAlternativeLanguages(playlist)

	for _, lang := range lang {
		var prev *m3u8.MediaItem
		for _, item := range playlist.Items {
			media, ok := item.(*m3u8.MediaItem)
			if !ok || media.Default == nil || media.Language == nil {
				continue
			}
			if *media.Default {
				prev = media
			}
			if *media.Language == lang {
				media.Default = &[]bool{true}[0]
			}
		}
		if prev == nil {
			continue
		}
		prev.Default = &[]bool{false}[0]
		name := fmt.Sprintf("%s_%s.m3u8", strings.ToLower(lang), suffix)
		if err := h.writePlaylist(filepath.Join(dir, name), playlist); err != nil {
			return err
		}
	}
	return nil
}

// return true if master has non-hd resolution playlists
func filterHDResolutions(playlist *m3u8.Playlist) bool {
	var items []m3u8.Item
	res := false
	for _, item := range playlist.Items {
		media, ok := item.(*m3u8.PlaylistItem)
		if !ok {
			items = append(items, item)
			continue
		}
		if media.Resolution != nil && media.Resolution.Width >= 1280 {
			continue
		}
		res = true
		items = append(items, item)
	}
	playlist.Items = items
	return res
}

// Beware special VOC hack:
// Invent missing codec information
func fixupCodecInformation(playlist *m3u8.Playlist) {
	for _, item := range playlist.Items {
		media, ok := item.(*m3u8.PlaylistItem)
		if !ok {
			continue
		}
		fmt.Printf("fixupCodecInformation: %s\n", media.String())
		if media.Codecs == nil {
			// Pretend we have H264 High level 4.0 video and AAC audio
			media.Codecs = &([]string{"avc1.640828,mp4a.40.2"}[0])
		}
	}
}

// get or create variant playlist
func (h *HLSParser) getVariantPlaylist(path string) (*VariantPlaylist, error) {
	if filepath.Dir(path) != h.basePath {
		return nil, fmt.Errorf("invalid playlist directory %s", path)
	}

	name := filepath.Base(path)
	sub, ok := h.subs[name]
	if !ok {
		sub = &VariantPlaylist{
			output: newLivePlaylist(h.playlistConfig.Size),
		}
		h.subs[name] = sub
	}
	return sub, nil
}

// compare sequence with last iteration and check for discontinuity
func (h *HLSParser) checkDiscontinuity(v *VariantPlaylist, source *m3u8.Playlist) {
	// if our segments went backwards
	if source.Sequence < v.lastSequence {
		res := v.output.applyDiscontinuity()
		switch deleted := res.(type) {
		case *m3u8.SegmentItem:
			h.expireFile(deleted.Segment)
		}
		v.lastIndex = 0
	} else {
		// only copy new segments
		diff := v.lastSequence - source.Sequence + source.SegmentSize() - v.lastSize
		if diff < 0 {
			v.lastIndex += diff
		}
	}
	v.lastSequence = source.Sequence
	v.lastSize = source.SegmentSize()
}

// append new items from source playlist
func (h *HLSParser) appendItems(v *VariantPlaylist, source *m3u8.Playlist) {
	index := 0
	for _, item := range source.Items {
		segment, ok := item.(*m3u8.SegmentItem)
		if !ok {
			continue
		}

		if index >= v.lastIndex {
			h.keepFile(segment.Segment)
			res := v.output.append(segment)
			switch deleted := res.(type) {
			case *m3u8.SegmentItem:
				h.expireFile(deleted.Segment)
			}
		}
		index++
	}
	v.lastIndex = index
}

// keep segment alive
func (h *HLSParser) keepFile(name string) {
	keep := make(chan struct{})
	path := filepath.Join(h.basePath, name)
	h.files[path] = keep
	h.registry.KeepFile(path, keep)
}

// allow segment to be deleted
func (h *HLSParser) expireFile(name string) {
	path := filepath.Join(h.basePath, name)
	if channel, ok := h.files[path]; ok {
		close(channel)
		delete(h.files, path)
	}
}

// write playlist to file
func (h *HLSParser) writePlaylist(path string, playlist *m3u8.Playlist) error {
	str, err := m3u8.Write(playlist)
	if err != nil {
		return err
	}
	err = h.writer.WriteFile(path, []byte(str))
	if err != nil {
		return err
	}
	h.keepFile(path)
	return nil
}

// get non-default languages from playlist
func getAlternativeLanguages(playlist *m3u8.Playlist) []string {
	var languages []string
	for _, item := range playlist.Items {
		media, ok := item.(*m3u8.MediaItem)
		if !ok || media.Default == nil || media.Language == nil {
			continue
		}
		if !*media.Default {
			languages = append(languages, *media.Language)
		}
	}
	return languages
}

type LivePlaylist struct {
	m3u8.Playlist
	size int
}

func newLivePlaylist(size int) *LivePlaylist {
	version := 3
	discontinuity := 0
	return &LivePlaylist{
		Playlist: m3u8.Playlist{
			Version:               &version,
			Live:                  true,
			DiscontinuitySequence: &discontinuity,
		},
		size: size,
	}
}

// returns removed item when max size is reached
func (lp *LivePlaylist) append(item m3u8.Item) (deleted m3u8.Item) {
	if len(lp.Items) == lp.size {
		deleted = lp.Items[0]
		switch deleted.(type) {
		case *m3u8.SegmentItem:
			lp.Sequence++
		case *m3u8.DiscontinuityItem:
			*lp.DiscontinuitySequence++
		}
		lp.Items = lp.Items[1:]
	}
	lp.Items = append(lp.Items, item)
	return
}

// returns removed item when max size is reached
func (lp *LivePlaylist) applyDiscontinuity() m3u8.Item {
	return lp.append(&m3u8.DiscontinuityItem{})
}

func (lp *LivePlaylist) setSegmentTarget(target int) {
	lp.Target = target
}
