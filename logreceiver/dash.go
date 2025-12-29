package logreceiver

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"github.com/zencoder/go-dash/mpd"
)

type Streams struct {
	cache sync.Map
}

type DashManifest struct {
	LastRefresh time.Time
	Qualities   map[string]string
}

func (s *Streams) getManifest(segmentPath string) DashManifest {
	base := filepath.Dir(segmentPath)
	val, ok := s.cache.LoadOrStore(base, DashManifest{Qualities: make(map[string]string)})
	if !ok {
		// not present in Cache getInBackground
		go func(b string) {
			err := s.getManifestInBackground(b)
			if err == nil {
				return
			}

			log.Println(err.Error())
			s.cache.Delete(b)
		}(base)
	}

	return val.(DashManifest)
}

func (s *Streams) getManifestInBackground(base string) error {
	path, err := url.JoinPath("http://127.0.0.1/", base, "manifest.mpd")

	log.Printf("try to get %s\n", path)

	if err != nil {
		return fmt.Errorf("failed to get manifest path: %s", err)
	}

	resp, err := http.Get(path)
	if err != nil {
		return fmt.Errorf("failed to get manifest: %s", err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get manifest: return status is %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to get manifest body: %s", err.Error())
	}

	manifest, err := mpd.ReadFromString(string(content))
	if err != nil {
		return fmt.Errorf("failed to decode manifest: %s", err.Error())
	}

	mD := DashManifest{}
	mD.Qualities = make(map[string]string)
	if len(manifest.Periods) != 1 {
		return fmt.Errorf("not exactly one period")
	}

	for _, a := range manifest.Periods[0].AdaptationSets {
		if a == nil {
			continue
		}

		if a.ContentType == nil || *a.ContentType != "video" {
			continue
		}

		for _, r := range a.Representations {
			if r == nil || r.Height == nil || r.ID == nil {
				continue
			}

			mD.Qualities[*r.ID] = fmt.Sprintf("%d", *r.Height)
		}
	}

	mD.LastRefresh = time.Now()
	s.cache.Store(base, mD)
	return nil
}

func (s *Streams) RefreshRoutine() {
	ticker := time.NewTicker(time.Minute * 30)
	for {
		<-ticker.C

		s.cache.Range(func(key, value any) bool {
			m := value.(DashManifest)
			base := key.(string)
			if time.Since(m.LastRefresh) < time.Hour {
				return true
			}

			err := s.getManifestInBackground(base)
			if err != nil {
				log.Println(err.Error())
				s.cache.Delete(base)
			}

			return true
		})
	}
}
