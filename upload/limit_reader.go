package upload

import (
	"errors"
	"io"
)
type LimitReader struct {
	R io.Reader
	N int64
}

func LimitReads(r io.Reader, limit int64) io.Reader {
	return &LimitReader{
		R: r,
		N: limit,
	}
}

var ErrLargeFile = errors.New("file too large")

func (l *LimitReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, ErrLargeFile
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return

}