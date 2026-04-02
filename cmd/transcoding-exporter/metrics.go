package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all prometheus metrics for transcoding jobs
type Metrics struct {
	reg prometheus.Registerer

	totalFrames           prometheus.CounterVec
	totalDroppedFrames    prometheus.CounterVec
	totalDuplicatedFrames prometheus.CounterVec
	totalBytes            prometheus.CounterVec
	totalDuration         prometheus.CounterVec
	currentBitrate        prometheus.GaugeVec
	currentFPS            prometheus.GaugeVec
	currentSpeed          prometheus.GaugeVec
}

// NewMetrics creates and registers all prometheus metrics
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		reg: reg,
		totalFrames: *promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "transcoding_frames_total",
			Help: "Current total frames processed for active transcoding job",
		}, []string{"stream_id"}),
		totalDroppedFrames: *promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "transcoding_frames_dropped_total",
			Help: "Current total frames dropped for active transcoding job",
		}, []string{"stream_id"}),
		totalDuplicatedFrames: *promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "transcoding_frames_duplicated_total",
			Help: "Current total frames duplicated for active transcoding job",
		}, []string{"stream_id"}),
		totalBytes: *promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "transcoding_bytes_total",
			Help: "Current total bytes written for active transcoding job",
		}, []string{"stream_id"}),
		totalDuration: *promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "transcoding_duration_seconds_total",
			Help: "Current total duration in seconds for active transcoding job",
		}, []string{"stream_id"}),
		currentBitrate: *promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "transcoding_bitrate_kbps",
			Help: "Current bitrate of active transcoding job",
		}, []string{"stream_id"}),
		currentFPS: *promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "transcoding_fps",
			Help: "Current frames per second of active transcoding job",
		}, []string{"stream_id"}),
		currentSpeed: *promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "transcoding_speed_factor",
			Help: "Current speed factor of active transcoding job (e.g. 1.0 = realtime, 2.0 = 2x faster)",
		}, []string{"stream_id"}),
	}
	return m
}

// RecordFrames increments the total frames counter
func (m *Metrics) RecordFrames(streamID string, count float64) {
	m.totalFrames.WithLabelValues(streamID).Add(count)
}

func (m *Metrics) RecordDroppedFrames(streamID string, count float64) {
	m.totalDroppedFrames.WithLabelValues(streamID).Add(count)
}

func (m *Metrics) RecordDuplicatedFrames(streamID string, count float64) {
	m.totalDuplicatedFrames.WithLabelValues(streamID).Add(count)
}

// RecordBytes increments the total bytes counter
func (m *Metrics) RecordBytes(streamID string, count float64) {
	m.totalBytes.WithLabelValues(streamID).Add(count)
}

func (m *Metrics) RecordDuration(streamID string, seconds float64) {
	m.totalDuration.WithLabelValues(streamID).Add(seconds)
}

// SetBitrate sets the current bitrate for a stream (in kbps)
func (m *Metrics) SetBitrate(streamID string, kbps float64) {
	m.currentBitrate.WithLabelValues(streamID).Set(kbps)
}

// SetFPS sets the current frames per second for a stream
func (m *Metrics) SetFPS(streamID string, fps float64) {
	m.currentFPS.WithLabelValues(streamID).Set(fps)
}

// SetSpeed sets the current speed factor for a stream (e.g. 1.0 = realtime, 2.0 = 2x faster)
func (m *Metrics) SetSpeed(streamID string, speed float64) {
	m.currentSpeed.WithLabelValues(streamID).Set(speed)
}

// DeleteStream removes all label combinations for a stream (called on job end)
func (m *Metrics) DeleteStream(streamID string) {
	m.totalFrames.DeleteLabelValues(streamID)
	m.totalDroppedFrames.DeleteLabelValues(streamID)
	m.totalDuplicatedFrames.DeleteLabelValues(streamID)
	m.totalBytes.DeleteLabelValues(streamID)
	m.totalDuration.DeleteLabelValues(streamID)
	m.currentBitrate.DeleteLabelValues(streamID)
	m.currentFPS.DeleteLabelValues(streamID)
	m.currentSpeed.DeleteLabelValues(streamID)
}
