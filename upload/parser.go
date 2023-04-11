package upload

import "io"

type Parser interface {
	ParsePlaylist(path string, playlist io.Reader) error
	Cleanup()
}

type ParserType string

const (
	ParserTypeHLS ParserType = "hls"
)
