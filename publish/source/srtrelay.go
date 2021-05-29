package source

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/voc/stream-api/config"
	"github.com/voc/stream-api/stream"
)

type SrtSource struct {
	conf config.SourceConfig
}

type srtStream struct {
	Name    string    `json:"name"`
	Clients int       `json:"clients"`
	Created time.Time `json:"created"`
	URL     string    `json:"url"`
}

// NewSrtSource creates a new goroutine parsing streams from an icecast2 server
func NewSrtrelayScraper(conf config.SourceConfig) *SrtSource {
	return &SrtSource{
		conf: conf,
	}
}

// scrape requests and parses data from the icecast server
func (srs SrtSource) Scrape(ctx context.Context) ([]*stream.Stream, error) {
	streams, err := srs.getStats(ctx, srs.conf.URL)
	if err != nil {
		return nil, err
	}
	return srs.mapSources(streams), nil
}

// getStats requests status data from icecast2
func (srs SrtSource) getStats(ctx context.Context, url string) ([]*srtStream, error) {
	req, err := http.NewRequest("GET", url+"/streams", nil)
	req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var stats []*srtStream
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return stats, nil
}

// mapSources parses a stream list from icecast2 status data
func (srs SrtSource) mapSources(srtStreams []*srtStream) []*stream.Stream {
	var streams []*stream.Stream
	for _, s := range srtStreams {
		streams = append(streams, &stream.Stream{
			Slug:   s.Name,
			Source: s.URL,
			Format: "mpegts",
		})
	}

	return streams
}
