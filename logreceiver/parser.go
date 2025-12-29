package logreceiver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"regexp"
	"slices"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/influxdata/go-syslog/v3/rfc3164"
	"github.com/prometheus/client_golang/prometheus"
)

type ParserConfig struct {
	// path to the syslog socket to listen on
	SocketPath string

	// duration of the sliding window for the counting
	SlidingWindow time.Duration

	// minimum number of segments a user must have watched to be counted
	MinSegments int

	StreamTimeout time.Duration

	// metrics registerer
	MetricsRegisterer prometheus.Registerer
}

type Parser struct {
	listener net.PacketConn
	hls      *HLSParser
	conf     ParserConfig

	segmentMutex sync.Mutex
	segments     map[UserId][]Segment

	// metrics
	viewers     map[StreamId]int
	metricMutex sync.Mutex
	lastCount   time.Time
}

// map user ip -> user_agent -> stream_name -> stream_id

type UserId struct {
	Ip      string
	Agent   string
	Country string
}

type Segment struct {
	Timestamp time.Time
	StreamId  StreamId
}

type StreamId struct {
	Slug      string
	Transport StreamTransport
	Type      StreamType
	Variant   StreamVariant
	Quality   StreamQuality
	Meta      string
	// EmbeddedAudio *StreamId
}

type StreamTransport string

const (
	StreamTransportHLS   StreamTransport = "hls"
	StreamTransportHLSLL StreamTransport = "hlsll"
	StreamTransportDASH  StreamTransport = "dash"
)

type StreamType string

const (
	StreamTypeVideo StreamType = "video"
	StreamTypeAudio StreamType = "audio"
)

type (
	StreamQuality string
	StreamVariant string
)

type CountryAndResolution struct {
	Country    string
	Resolution string
}

type StreamRepresentation struct {
	Type    string
	Name    string
	Quality string
}

type NginxLogEntry struct {
	Address string `json:"remote_addr"`
	Method  string `json:"method"`

	Uri    string `json:"uri"`
	Status string `json:"status"`

	BytesSent string `json:"bytes_sent"`

	// address of upstream relay
	Upstream string `json:"upstream"`

	// used for metrics
	Protocol    string `json:"protocol"`
	Scheme      string `json:"scheme"`
	SSLCipher   string `json:"ssl_cipher"`
	SSLProtocol string `json:"ssl_protocol"`
	UserAgent   string `json:"user_agent"`
	Country     string `json:"country"`
}

func NewParser(ctx context.Context, conf ParserConfig) (*Parser, error) {
	_ = os.RemoveAll(conf.SocketPath)
	unix.Umask(0)
	listener, err := net.ListenPacket("unixgram", conf.SocketPath)
	if err != nil {
		return nil, err
	}
	unix.Umask(0o022)
	slog.Info("logparser listening on", "address", listener.LocalAddr(), "config", conf)
	p := &Parser{
		listener: listener,
		conf:     conf,
		hls:      NewHLSParser(ctx, conf.StreamTimeout),
		segments: make(map[UserId][]Segment),
		viewers:  make(map[StreamId]int),
	}
	err = conf.MetricsRegisterer.Register(p)
	if err != nil {
		return nil, err
	}
	go p.run(ctx)
	return p, nil
}

func (p *Parser) run(ctx context.Context) {
	sysParser := rfc3164.NewParser()
	buf := make([]byte, 16384)
	for {
		if ctx.Err() != nil {
			return
		}
		err := p.listener.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		if err != nil {
			slog.Error("failed to set read deadline", "err", err)
		}
		n, _, err := p.listener.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			slog.Error("failed to read", "err", err)
			continue
		}

		m, err := sysParser.Parse(buf[0:n])
		if err != nil {
			slog.Error("failed to parse", "err", err)
			continue
		}

		parsed, ok := m.(*rfc3164.SyslogMessage)
		if !ok || parsed == nil || parsed.Message == nil {
			slog.Warn("got invalid syslog msg")
			continue
		}

		var point NginxLogEntry
		err = json.Unmarshal([]byte(*parsed.Message), &point)
		if err != nil {
			slog.Error("failed to unmarshal json", "err", err)
			continue
		}

		p.parseEntry(point)
		// slog.Debug("parsed", "entry", point)
	}
}

var uriRegexp = regexp.MustCompile(`\/(dash|hls|hlsll)\/([^\/]+)\/(.+)$`)

func (p *Parser) parseEntry(point NginxLogEntry) {
	if point.Method != http.MethodGet {
		return
	}

	groups := uriRegexp.FindStringSubmatch(point.Uri)
	if groups == nil {
		slog.Warn("regex doesn't match", "url", point.Uri)
		return
	}

	streamTransport := groups[1]
	slug := groups[2]
	file := groups[3]
	var stream *StreamId
	var ok bool

	// slog.Debug("match", "transport", streamTransport, "slug", slug, "file", file, "event", point)

	switch streamTransport {
	case "dash":
		return
	case "hlsll":
		fallthrough
	case "hls":
		stream, ok = p.hls.GetQuality(slug, StreamTransport(streamTransport), file)
	default:
		return
	}
	slog.Debug("entry", "transport", streamTransport, "slug", slug, "file", file, "stream", stream, "ok", ok)

	if !ok {
		return
	}
	userId := UserId{point.Address, point.UserAgent, point.Country}
	p.addSegment(userId, stream)
}

// addSegment adds a segment to the user counter
func (p *Parser) addSegment(userId UserId, stream *StreamId) {
	p.segmentMutex.Lock()
	defer p.segmentMutex.Unlock()
	segments := p.segments[userId]
	segments = append(segments, Segment{
		Timestamp: time.Now(),
		StreamId:  *stream,
	})
	p.segments[userId] = segments
}

// descriptors.
func (p *Parser) Describe(ch chan<- *prometheus.Desc) {
	ch <- viewersDesc
}

// Collect first triggers the ReallyExpensiveAssessmentOfTheSystemState. Then it
// creates constant metrics for each host on the fly based on the returned data.
//
// Note that Collect could be called concurrently, so we depend on
// ReallyExpensiveAssessmentOfTheSystemState to be concurrency-safe.
func (p *Parser) Collect(ch chan<- prometheus.Metric) {
	p.metricMutex.Lock()
	defer p.metricMutex.Unlock()
	// only count every 3 seconds
	if time.Since(p.lastCount) > time.Second*3 {
		slog.Debug("counting")
		p.count()
	}
	for s, count := range p.viewers {
		ch <- prometheus.MustNewConstMetric(viewersDesc, prometheus.GaugeValue, float64(count),
			s.Slug, string(s.Type), string(s.Quality), s.Meta, string(s.Transport), string(s.Variant))
	}
}

var viewersDesc = prometheus.NewDesc(
	"viewers",
	"Current viewer count on a specific stream on this relay.",
	[]string{"slug", "type", "quality", "meta", "transport", "variant"}, nil,
)

// var countcountryDesc = prometheus.NewDesc(
// 	"viewers_country",
// 	"Current viewer per country on this relay.",
// 	[]string{"country", "resolution"}, nil,
// )

// count currently active users and their stream representation
func (p *Parser) count() {
	p.lastCount = time.Now()
	// reset count
	p.viewers = make(map[StreamId]int)
	p.segmentMutex.Lock()
	defer p.segmentMutex.Unlock()
	for userId, segments := range p.segments {
		videoStreams := make(map[StreamId]int)
		audioStreams := make(map[StreamId]int)
		removeSegments := 0
		for i, segment := range segments {
			// only consider segments that are not older than the sliding window
			if time.Since(segment.Timestamp) > p.conf.SlidingWindow {
				removeSegments = i + 1
				continue
			}
			if segment.StreamId.Type == StreamTypeVideo {
				videoStreams[segment.StreamId] += 1
			} else {
				audioStreams[segment.StreamId] += 1
			}
		}

		if removeSegments == len(segments) {
			slog.Debug("expire", "user", userId)
			delete(p.segments, userId)
			continue
		}
		// trim old segments
		p.segments[userId] = slices.Delete(segments, 0, removeSegments)

		// account for the most watched audio and video stream
		videoStream, videoOk := p.maxQuality(videoStreams)
		if videoOk {
			p.viewers[videoStream]++
		}
		audioStream, audioOk := p.maxQuality(audioStreams)
		if audioOk {
			p.viewers[audioStream]++
		}

		// if the most watched video stream has audio embedded and we have no separate audio for this user, count it as well
		// if videoOk && videoStream.EmbeddedAudio != nil && !audioOk {
		// 	p.viewers[*videoStream.EmbeddedAudio]++
		// }
	}
}

func (p *Parser) maxQuality(qualities map[StreamId]int) (StreamId, bool) {
	max := 0
	maxQuality := StreamId{}
	for quality, count := range qualities {
		if count > max {
			max = count
			maxQuality = quality
		}
	}
	return maxQuality, max > p.conf.MinSegments
}
