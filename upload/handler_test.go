package upload

import (
	"io"
	"log"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

// 3:40PM DBG handle method=PUT url=/hls/s1/SD.m3u8
// 3:40PM DBG handle method=PUT url=/hls/s1/Native_1692106828_0006.ts
// 3:40PM DBG handle method=PUT url=/hls/s1/Native.m3u8
// 3:40PM DBG handle method=PUT url=/thumbnail/s1/poster.jpeg
// 3:40PM DBG handle method=PUT url=/thumbnail/s1/thumb.jpeg
// 3:40PM DBG handle method=PUT url=/hls/s1/Translated-2_1692106828_0006.ts
// 3:40PM DBG handle method=PUT url=/hls/s1/Translated-2.m3u8
// 3:40PM DBG handle method=PUT url=/hls/s1/Translated_1692106828_0006.ts
// 3:40PM DBG handle method=PUT url=/hls/s1/Translated.m3u8
// 3:40PM DBG handle method=PUT url=/hls/s1/Slides_1692106831_0006.ts

type MockWriter struct{}

func (m MockWriter) CopyFile(path string, src io.Reader) error {
	return nil
}

func TestHandler(t *testing.T) {
	h := &Handler{
		copier:   MockWriter{},
		registry: NewFileRegistry(FileRegistryConfig{}),
		store: NewStreamStore(StreamStoreConfig{
			StreamTimeout: time.Millisecond * 100,
		}),
	}
	defer h.Stop()
	h.Validate("s1", "hls/s1/SD.m3u8", "source1")
	assert.NilError(t, h.HandleFile(nil, "s1", "thumbnail/s1/poster.jpeg"))
	assert.NilError(t, h.HandleFile(nil, "s1", "thumbnail/s1/thumb.jpeg"))
	time.Sleep(time.Millisecond * 50)

	res, ok := h.registry.files["thumbnail/s1/thumb.jpeg"]
	assert.Assert(t, ok)
	assert.Assert(t, res != nil)
	log.Println(ok, *res)
	assert.Assert(t, res.keep != nil)
}
