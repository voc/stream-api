package source

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/voc/stream-api/config"
	"github.com/voc/stream-api/stream"
)

type IcecastSource struct {
	conf config.SourceConfig
}

type IcecastStream struct {
	Listeners    int     `json:"listeners"`
	ListenerPeak int     `json:"listener_peak"`
	ServerType   string  `json:"server_type"`
	StreamStart  IsoTime `json:"stream_start_iso8601"`
	URL          string  `json:"listenurl"`
}

// IcecastStats represents the format of the icecast2 json stats api
type IcecastStats struct {
	Stats struct {
		Source json.RawMessage `json:"source,omitempty"`
	} `json:"icestats"`
}

type IsoTime struct {
	time.Time
}

// UnmarshalJSON parses a Iso8601 Date from a string format in json
func (it *IsoTime) UnmarshalJSON(input []byte) error {
	strInput := strings.Trim(string(input), `"`)
	newTime, err := time.Parse("2006-01-02T15:04:05-0700", strInput)
	if err != nil {
		return err
	}
	it.Time = newTime
	return nil
}

// NewIcecastSource creates a new goroutine parsing streams from an icecast2 server
func NewIcecastScraper(conf config.SourceConfig) *IcecastSource {
	return &IcecastSource{
		conf: conf,
	}
}

// scrape requests and parses data from the icecast server
func (ics IcecastSource) Scrape(ctx context.Context) ([]*stream.Stream, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", ics.conf.URL+"/status-json.xsl", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "get")
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read")
	}

	streams, err := ics.parse(data)
	if err != nil {
		return nil, errors.Wrap(err, "parse")
	}
	return ics.mapStreams(streams), nil
}

func (ics IcecastSource) parse(data []byte) ([]*IcecastStream, error) {
	stats := IcecastStats{}
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, errors.Wrap(err, "unmarshal stats")
	}

	var iceStreams []*IcecastStream
	if stats.Stats.Source == nil {
		return iceStreams, nil
	}

	// try to unmarshal slice
	err := json.Unmarshal(stats.Stats.Source, &iceStreams)
	if err == nil {
		return iceStreams, nil
	}

	// try to unmarshal single source
	iceStream := IcecastStream{}
	err = json.Unmarshal(stats.Stats.Source, &iceStream)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal stream")
	}
	iceStreams = append(iceStreams, &iceStream)
	return iceStreams, nil
}

func (ics IcecastSource) mapStreams(iceStreams []*IcecastStream) []*stream.Stream {
	var streams []*stream.Stream
	for _, source := range iceStreams {
		slug := path.Base(source.URL)
		streams = append(streams, &stream.Stream{
			Format: "matroska", // fixed for video only for now
			Slug:   slug,
			Source: source.URL,
		})
	}

	return streams
}
