package upload

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

type HLSMetrics struct {
	reg                           prometheus.Registerer
	slug                          string
	mutex                         sync.Mutex
	targetDuration                int
	segmentDurations              map[string][]float64 // playlist name -> list of segment duration differences
	totalInvalidPlaylistDurations *prometheus.CounterVec
	maxDesc                       *prometheus.Desc
	meanDesc                      *prometheus.Desc
}

func NewHLSMetrics(slug string, reg prometheus.Registerer) *HLSMetrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	constLabels := prometheus.Labels{
		"slug": slug,
	}
	m := &HLSMetrics{
		reg:              reg,
		slug:             slug,
		segmentDurations: make(map[string][]float64),
		totalInvalidPlaylistDurations: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name:        "upload_total_invalid_playlist_durations",
			Help:        "Total number of playlists received with invalid target duration (not matching other variant playlists)",
			ConstLabels: constLabels,
		}, []string{"playlist_name"}),
		maxDesc: prometheus.NewDesc(
			"playlist_duration_diff_max_seconds",
			"Maximum difference between HLS targetDuration and actual upload interval",
			[]string{"playlist_name"}, constLabels),
		meanDesc: prometheus.NewDesc(
			"playlist_duration_diff_mean_seconds",
			"Mean difference between HLS targetDuration and actual upload interval",
			[]string{"playlist_name"}, constLabels),
	}
	reg.MustRegister(m)
	return m
}

func (m *HLSMetrics) Unregister() {
	m.reg.Unregister(m.totalInvalidPlaylistDurations)
	m.reg.Unregister(m)
}

func (m *HLSMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- m.maxDesc
	ch <- m.meanDesc
}

func (m *HLSMetrics) Collect(ch chan<- prometheus.Metric) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.targetDuration == 0 {
		return
	}
	for playlistName, durations := range m.segmentDurations {
		if len(durations) == 0 {
			continue
		}
		max := 0.0
		sum := 0.0
		for _, d := range durations {
			diff := abs(d - float64(m.targetDuration))
			sum += diff
			if diff > max {
				max = diff
			}
		}
		ch <- prometheus.MustNewConstMetric(m.maxDesc, prometheus.GaugeValue, max, playlistName)
		ch <- prometheus.MustNewConstMetric(m.meanDesc, prometheus.GaugeValue, sum/float64(len(durations)), playlistName)
	}
}

func (m *HLSMetrics) RecordTargetDuration(targetDuration int, playlist string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.targetDuration == 0 {
		m.targetDuration = targetDuration
	}
	if m.targetDuration != targetDuration {
		log.Warn().
			Str("slug", m.slug).
			Int("targetDuration", targetDuration).
			Int("lastDuration", m.targetDuration).
			Str("playlist", playlist).
			Msg("invalid target duration in playlist")
		m.totalInvalidPlaylistDurations.WithLabelValues(playlist).Inc()
		m.targetDuration = targetDuration
	}
}

func (m *HLSMetrics) RecordSegmentDuration(playlistName string, duration float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	durations, ok := m.segmentDurations[playlistName]
	if !ok {
		durations = []float64{}
	}
	m.segmentDurations[playlistName] = append(durations, duration)
	if len(m.segmentDurations[playlistName]) > 10 {
		m.segmentDurations[playlistName] = m.segmentDurations[playlistName][1:]
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
