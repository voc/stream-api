package upload

import (
	"errors"
	"io"
	"time"

	"github.com/quangngotan95/go-m3u8/m3u8"
	"github.com/zencoder/go-dash/mpd"
)

type Parser struct{}

var (
	errNoSegmentTemplate = errors.New("manifest loader requires Representations with SegmentTemplate for now")
	errTimescaleMissing  = errors.New("manifest SegmentTemplate is missing timescale")
)

// parse segment interval from hls playlist
func (p *Parser) parseHLSPlaylist(reader io.Reader) (time.Duration, error) {
	playlist, err := m3u8.Read(reader)
	if err != nil {
		return 0, err
	}

	total := float64(0)
	count := 0

	for _, item := range playlist.Items {
		if segment, ok := (item).(*m3u8.SegmentItem); ok {
			total += segment.Duration
			count++
		}
	}
	// no items yet or master playlist
	if count == 0 {
		return 0, nil
	}
	avg := (time.Second * time.Duration(total)) / time.Duration(count)

	return avg, nil
}

// parse update interval from dash manifest
func (p *Parser) parseDashManifest(reader io.Reader) (time.Duration, error) {
	manifest, err := mpd.Read(reader)
	if err != nil {
		return 0, err
	}

	interval := time.Second * 10
	if manifest.MinimumUpdatePeriod != nil {
		minUpdate, err := mpd.ParseDuration(*manifest.MinimumUpdatePeriod)
		if err == nil && minUpdate > 0 {
			interval = minUpdate
		}
	}

	return interval, nil
}
