package upload

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"github.com/quangngotan95/go-m3u8/m3u8"
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

func expectPlaylist(t *testing.T, playlist *m3u8.Playlist, path string) {
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Error(err)
	}
	expected := string(buf)
	actual, err := m3u8.Write(playlist)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("playlists are not equal, expected:\n`%v`\n, got:\n`%v`\n", expected, actual)
	}
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
	expectPlaylist(t, playlist, "fixtures/with_codec.m3u8")
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
	expectPlaylist(t, playlist, "fixtures/native_sd.m3u8")
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
