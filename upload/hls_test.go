package upload

import (
	"bytes"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/quangngotan95/go-m3u8/m3u8"
	"gotest.tools/v3/assert"
)

// AssertEqual checks if values are equal
func AssertEqual(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		return
	}
	// debug.PrintStack()
	t.Errorf("Received %v (type %v), expected %v (type %v)", a, reflect.TypeOf(a), b, reflect.TypeOf(b))
}

func readPlaylist(t *testing.T, path string) *m3u8.Playlist {
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Error(err)
	}
	reader := bytes.NewBuffer(buf)
	playlist, err := m3u8.Read(reader)
	if err != nil {
		t.Error(err)
	}
	return playlist
}

func parsePlaylist(t *testing.T, data string) *m3u8.Playlist {
	reader := bytes.NewBufferString(data)
	playlist, err := m3u8.Read(reader)
	if err != nil {
		t.Error(err)
	}
	return playlist
}

func expectPlaylist(t *testing.T, playlist *m3u8.Playlist, expected string, msg string) {
	actual, err := m3u8.Write(playlist)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("%s: expected:\n`%v`\n, got:\n`%v`\n", msg, expected, actual)
	}
}

func expectPlaylistFile(t *testing.T, playlist *m3u8.Playlist, path string, msg string) {
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Error(err)
	}
	expected := string(buf)
	expectPlaylist(t, playlist, expected, msg)
}

func Test_getAlternativeLanguages(t *testing.T) {
	playlist := readPlaylist(t, "fixtures/native_hd.m3u8")
	if !playlist.IsMaster() {
		t.Error("playlist is not master")
	}

	expected := []string{"Translated", "Translated-2"}
	langs := getAlternativeLanguages(playlist)
	if !reflect.DeepEqual(expected, langs) {
		t.Errorf("getAlternativeLanguages() = %v, want %v", langs, expected)
	}
}

func Test_fixupCodecInformation(t *testing.T) {
	playlist := readPlaylist(t, "fixtures/no_codec.m3u8")
	if !playlist.IsMaster() {
		t.Error("playlist is not master")
	}

	fixupCodecInformation(playlist)
	expectPlaylistFile(t, playlist, "fixtures/with_codec.m3u8", "playlist should contain codec information")
}

func Test_filterHDResolutions(t *testing.T) {
	playlist := readPlaylist(t, "fixtures/native_hd.m3u8")
	if !playlist.IsMaster() {
		t.Error("playlist is not master")
	}

	res := filterHDResolutions(playlist)
	if !res {
		t.Errorf("filterHDResolutions() = %v, want %v", res, true)
	}
	expectPlaylistFile(t, playlist, "fixtures/native_sd.m3u8", "playlist should not contain HD resolutions")
}

func Test_playlistAppend(t *testing.T) {
	p := newLivePlaylist(3)
	p.append(&m3u8.DiscontinuityItem{})
	for i := 0; i < 3; i++ {
		p.append(&m3u8.SegmentItem{})
	}
	AssertEqual(t, *p.DiscontinuitySequence, 1)
	AssertEqual(t, p.Sequence, 0)
	p.append(&m3u8.SegmentItem{})
	AssertEqual(t, p.Sequence, 1)
}

// If the sequence went backward a discontinuity should be inserted
func Test_VariantSequenceRollback(t *testing.T) {
	h := newHLSParser(HLSConfiguration{
		slug:     "foo",
		basePath: "/foo",
		playlistConfig: PlaylistConfig{
			Size: 3,
		},
		writer:   nil,
		registry: NewFileRegistry(FileRegistryConfig{ExpireInterval: time.Second, KeepDelay: time.Second}),
	})
	one, err := h.processVariant("/foo/foo.m3u8", parsePlaylist(t, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:3
#EXT-X-MEDIA-SEQUENCE:85
#EXTINF:2.986,
segment_Native85.ts
#EXTINF:3.008,
segment_Native86.ts`))
	assert.NilError(t, err)

	// expect playlist to start at discont 0
	expectPlaylist(t, one, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-DISCONTINUITY-SEQUENCE:0
#EXT-X-TARGETDURATION:3
#EXTINF:2.986,
segment_Native85.ts
#EXTINF:3.008,
segment_Native86.ts
`, "playlist should be unchanged except sequence")

	two, err := h.processVariant("/foo/foo.m3u8", parsePlaylist(t, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:3
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:3,
segment_Native1.ts
#EXTINF:3,
segment_Native2.ts`))
	assert.NilError(t, err)

	// expect playlist to start at discont 0
	expectPlaylist(t, two, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-MEDIA-SEQUENCE:2
#EXT-X-DISCONTINUITY-SEQUENCE:0
#EXT-X-TARGETDURATION:3
#EXT-X-DISCONTINUITY

#EXTINF:3,
segment_Native1.ts
#EXTINF:3,
segment_Native2.ts
`, "playlist should have a discontinuity")
}

// If the playlist suddenly got shorter but the sequence is the same we still reset the playlist
func Test_VariantSequenceShorterPlaylist(t *testing.T) {
	h := newHLSParser(HLSConfiguration{
		slug:     "foo",
		basePath: "/foo",
		playlistConfig: PlaylistConfig{
			Size: 3,
		},
		writer:   nil,
		registry: NewFileRegistry(FileRegistryConfig{ExpireInterval: time.Second, KeepDelay: time.Second}),
	})
	one, err := h.processVariant("/foo/foo.m3u8", parsePlaylist(t, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:3
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:3.000,
segment_Native01.ts
#EXTINF:3.000,
segment_Native02.ts
#EXTINF:3.000,
segment_Native03.ts
#EXTINF:3.000,
segment_Native04.ts`))
	assert.NilError(t, err)

	// expect playlist to start at discont 0
	expectPlaylist(t, one, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-MEDIA-SEQUENCE:1
#EXT-X-DISCONTINUITY-SEQUENCE:0
#EXT-X-TARGETDURATION:3
#EXTINF:3,
segment_Native02.ts
#EXTINF:3,
segment_Native03.ts
#EXTINF:3,
segment_Native04.ts
`, "playlist should be unchanged")

	two, err := h.processVariant("/foo/foo.m3u8", parsePlaylist(t, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:3
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:3.000,
segment_Native01.ts
#EXTINF:3.000,
segment_Native02.ts`))
	assert.NilError(t, err)

	// expect playlist to start at discont 0
	expectPlaylist(t, two, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-MEDIA-SEQUENCE:4
#EXT-X-DISCONTINUITY-SEQUENCE:0
#EXT-X-TARGETDURATION:3
#EXT-X-DISCONTINUITY

#EXTINF:3,
segment_Native01.ts
#EXTINF:3,
segment_Native02.ts
`, "playlist should have a discontinuity")

	three, err := h.processVariant("/foo/foo.m3u8", parsePlaylist(t, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:3
#EXT-X-MEDIA-SEQUENCE:2
#EXTINF:3.000,
segment_Native03.ts
#EXTINF:3.000,
segment_Native04.ts`))
	assert.NilError(t, err)

	// expect playlist to start at discont 0
	expectPlaylist(t, three, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-MEDIA-SEQUENCE:5
#EXT-X-DISCONTINUITY-SEQUENCE:1
#EXT-X-TARGETDURATION:3
#EXTINF:3,
segment_Native02.ts
#EXTINF:3,
segment_Native03.ts
#EXTINF:3,
segment_Native04.ts
`, "should continue to append")
}
