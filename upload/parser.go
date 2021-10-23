package upload

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/quangngotan95/go-m3u8/m3u8"
	"github.com/zencoder/go-dash/mpd"
)

type Parser struct{}

var (
	errAvailabilityStartMissing = errors.New("manifest missing AvailabilityStartTime")
	errNoPresentationDelay      = errors.New("manifest missing SuggestedPresentationDelay")
	errNoSegmentTemplate        = errors.New("manifest loader requires Representations with SegmentTemplate for now")
	errTimescaleMissing         = errors.New("manifest SegmentTemplate is missing timescale")
)

// parseMpd parses m3u8 playlists and creates download tasks for all segments.
// Can work with multi-quality master-playlists.
func (p *Parser) parseHLSPlaylist(reader io.Reader, basePath string) ([]string, time.Duration, error) {
	playlist, err := m3u8.Read(reader)
	if err != nil {
		return nil, 0, err
	}

	total := float64(0)
	count := float64(0)

	var segments []string
	for _, item := range playlist.Items {
		if segment, ok := (item).(*m3u8.SegmentItem); ok {
			total += segment.Duration
			segments = append(segments, filepath.Join(basePath, segment.Segment))
			count++
		}
	}
	if count == 0 {
		return nil, 0, errors.New("parsed playlist without items")
	}
	avg := (time.Second * time.Duration(total)) / time.Duration(count)

	return segments, avg, nil
}

func (p *Parser) parseDashManifest(reader io.Reader, basePath string) ([]string, time.Duration, error) {
	manifest, err := mpd.Read(reader)
	if err != nil {
		return nil, 0, err
	}

	interval := time.Second * 10
	if manifest.MinimumUpdatePeriod != nil {
		minUpdate, err := mpd.ParseDuration(*manifest.MinimumUpdatePeriod)
		if err == nil && minUpdate > 0 {
			interval = minUpdate
		}
	}

	var segments []string
	period := manifest.Periods[0]
	for _, as := range period.AdaptationSets {
		for _, representation := range as.Representations {

			// Just support SegmentTimeline for now
			if representation.SegmentTemplate == nil {
				return nil, 0, errNoSegmentTemplate
			}
			template := representation.SegmentTemplate
			if template.Timescale == nil {
				return nil, 0, errTimescaleMissing
			}
			// timescale := *template.Timescale
			timeline := template.SegmentTimeline
			number := int64(0)
			if template.StartNumber != nil {
				number = *template.StartNumber
			}

			// add init segment
			if template.Initialization != nil {
				name := dashSegmentName(*template.Initialization, representation, 0)
				segments = append(segments, filepath.Join(basePath, name))
			}

			for _, segment := range timeline.Segments {
				repeat := 0
				if segment.RepeatCount != nil {
					repeat = *segment.RepeatCount
				}

				for n := 0; n < repeat+1; n++ {
					name := dashSegmentName(*template.Media, representation, number)
					segments = append(segments, filepath.Join(basePath, name))
					number++
				}
			}
		}
	}

	return segments, interval, nil
}

// dashSegmentName templates a MPEG-DASH SegmentTemplate name
func dashSegmentName(template string, r *mpd.Representation, offset int64) string {
	st := r.SegmentTemplate
	id := *r.ID
	number := *st.StartNumber + offset
	name := strings.ReplaceAll(template, "$RepresentationID$", id)

	return strings.ReplaceAll(name, "$Number$", strconv.FormatInt(number, 10))
}

// getSubURL returns the URL to a playlist entry
func (p *Parser) getSubURL(playlistURL *url.URL, subURI string) (subURL *url.URL, err error) {
	var str string
	if strings.HasPrefix(subURI, "/") {
		// absolute subURI
		str = fmt.Sprintf("%s://%s%s", playlistURL.Scheme, playlistURL.Host, subURI)
	} else {
		// relative subURI
		str = fmt.Sprintf("%s://%s%s/%s", playlistURL.Scheme, playlistURL.Host, path.Dir(playlistURL.Path), subURI)
	}
	subURL, err = url.Parse(str)
	return
}
