package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

// ProgressHandler handles HTTP requests containing FFmpeg progress data
type ProgressHandler struct {
	metrics *Metrics
}

// activeStream tracks metrics for a currently-transcoding stream
type activeStream struct {
	streamID     string
	lastFrame    float64
	lastDropped  float64
	lastDup      float64
	lastBytes    float64
	lastDuration float64
}

// NewProgressHandler creates a new progress handler
func NewProgressHandler(metrics *Metrics) *ProgressHandler {
	return &ProgressHandler{
		metrics: metrics,
	}
}

// ServeHTTP handles incoming progress data from FFmpeg
// Expected path: /progress/{stream_id}
// Body contains key=value pairs (one per line)
func (h *ProgressHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		slog.Warn("invalid method for progress handler", "method", r.Method)
		return
	}

	// Extract stream_id from path: /progress/{stream_id}
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/progress/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.Error(w, "stream_id required in path", http.StatusBadRequest)
		slog.Warn("progress request missing stream_id", "path", r.URL.Path)
		return
	}

	streamID := pathParts[0]
	remoteAddr := r.RemoteAddr
	slog.Info("progress request received", "stream_id", streamID, "remote_addr", remoteAddr)

	// Parse progress data from request body
	scanner := bufio.NewScanner(r.Body)
	defer r.Body.Close()

	defer h.metrics.DeleteStream(streamID)
	stream := activeStream{
		streamID: streamID,
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		slog.Debug("progress line", "stream_id", streamID, "text", line)

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			continue
		}
		h.updateMetrics(&stream, key, floatVal)
	}

	if err := scanner.Err(); err != nil {
		slog.Warn("error reading progress data", "stream_id", streamID, "err", err)
		http.Error(w, "error reading body", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprintf(w, "OK\n")
}

// updateMetrics updates prometheus metrics based on FFmpeg progress data
func (h *ProgressHandler) updateMetrics(stream *activeStream, key string, value float64) {
	switch key {
	case "frame":
		// Track frames processed
		h.record(&stream.lastFrame, value, h.metrics.RecordFrames, stream.streamID)
	case "drop_frames":
		// Track dropped frames
		h.record(&stream.lastDropped, value, h.metrics.RecordDroppedFrames, stream.streamID)
	case "dup_frames":
		// Track duplicated frames
		h.record(&stream.lastDup, value, h.metrics.RecordDuplicatedFrames, stream.streamID)
	case "total_size":
		// Track bytes written
		h.record(&stream.lastBytes, value, h.metrics.RecordBytes, stream.streamID)
	case "out_time_us":
		// Convert microseconds to seconds and track duration
		durationSeconds := value / 1e6
		h.record(&stream.lastDuration, durationSeconds, h.metrics.RecordDuration, stream.streamID)
	case "bitrate":
		// FFmpeg reports bitrate in kbps, already suitable for prometheus
		h.metrics.SetBitrate(stream.streamID, value)
	case "fps":
		// Frames per second
		h.metrics.SetFPS(stream.streamID, value)
	}
}

func (h *ProgressHandler) record(state *float64, newvalue float64, recordFunc func(string, float64), streamID string) {
	if newvalue > *state {
		delta := newvalue - *state
		recordFunc(streamID, delta)
		*state = newvalue
	}
}
