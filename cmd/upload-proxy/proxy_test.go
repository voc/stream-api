package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Proxy upload smoke test
func TestUpload(t *testing.T) {
	ch := make(chan *http.Request, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		ch <- r
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sink, err := NewSink(SinkConfig{
		Address: srv.URL + "/upload",
	})
	if err != nil {
		t.Fatal(err)
	}

	proxy, err := NewProxy(ctx, "127.0.0.1:0", []*Sink{sink})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		select {
		case <-ctx.Done():
		case err := <-proxy.Errors():
			t.Error(err)
		}
	}()
	rd := io.LimitReader(rand.Reader, 1024*1024*10)
	_, err = http.Post(fmt.Sprintf("http://%s/test.mp4", proxy.Address()), "text/plain", rd)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-ctx.Done():
		t.Fatal("timeout")
	case r := <-ch:
		if r.URL.Path != "/upload/test.mp4" {
			t.Errorf("invalid path: %s", r.URL.Path)
		}
	}
}
