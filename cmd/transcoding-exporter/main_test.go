package main

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	os.Exit(m.Run())
}

func TestProgressHandlerHTTP(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewMetrics(reg)
	handler := NewProgressHandler(metrics)

	testData := []byte("frame=100\ntotal_size=1000000\nbitrate=2500.0k\nfps=29.97\nout_time_us=5000000\nframe=200\ntotal_size=2000000\nbitrate=2400.0k\nprogress=end\n")
	req := httptest.NewRequest(http.MethodPost, "/progress/test-stream-1", bytes.NewReader(testData))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "OK\n" {
		t.Fatalf("expected response 'OK\\n', got '%s'", w.Body.String())
	}
}

func TestMetricsUpdated(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewMetrics(reg)
	handler := NewProgressHandler(metrics)

	rd, wr := io.Pipe()
	defer wr.Close()
	testData := []byte("frame=100\nframe=200\n")
	req := httptest.NewRequest(http.MethodPost, "/progress/test-stream", rd)
	w := httptest.NewRecorder()

	go func() {
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	}()
	_, err := wr.Write(testData)
	if err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}
	time.Sleep(time.Millisecond * 100)
	snapshot := snapshotMetrics(t, reg, "test-stream")
	expected := metricSnapshot{
		framesTotal:   200, // Frame count from the update
		bytesTotal:    0,   // No bytes update in this test
		durationTotal: 0,   // No duration update in this test
	}
	if !reflect.DeepEqual(snapshot, expected) {
		t.Fatalf("metrics snapshot mismatch:\nexpected: %+v\ngot: %+v", expected, snapshot)
	}
}

type metricSnapshot struct {
	framesTotal   float64
	bytesTotal    float64
	durationTotal float64
}

func snapshotMetrics(t *testing.T, reg prometheus.Gatherer, streamID string) metricSnapshot {
	mf, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather failed: %v", err)
	}

	s := metricSnapshot{}
	for _, family := range mf {
		fmt.Println("Family:", family.GetName())
		for _, m := range family.Metric {
			var streamLabel string
			for _, lp := range m.Label {
				if lp.GetName() == "stream_id" {
					streamLabel = lp.GetValue()
					break
				}
			}
			if streamLabel != streamID {
				continue
			}

			switch family.GetName() {
			case "transcoding_frames_total":
				s.framesTotal = m.Counter.GetValue()
			case "transcoding_bytes_total":
				s.bytesTotal = m.Counter.GetValue()
			case "transcoding_duration_seconds_total":
				s.durationTotal = m.Counter.GetValue()
			}
		}
	}
	return s
}
