package upload

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type FileWriter interface {
	// writes data to path
	WriteFile(path string, data []byte) error
}

type FileCopier interface {
	// copies file from reader into path
	CopyFile(path string, src io.Reader) error
}

type AtomicWriter struct{}

func (w AtomicWriter) open(path string) (io.WriteCloser, string, error) {
	dir := filepath.Dir(path)
	tmpPath := path + ".tmp"
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return nil, "", fmt.Errorf("%w", err)
	}

	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0o644)
	if err != nil {
		return nil, "", fmt.Errorf("open: %w", err)
	}

	return file, tmpPath, nil
}

func (w AtomicWriter) write(file io.WriteCloser, data []byte) (err error) {
	_, err = file.Write(data)
	if err1 := file.Close(); err1 != nil && err == nil {
		err = err1
	}
	return
}

func (w AtomicWriter) copy(file io.WriteCloser, src io.Reader) (err error) {
	_, err = io.Copy(file, src)
	if err1 := file.Close(); err1 != nil && err == nil {
		err = err1
	}
	return
}

func (w AtomicWriter) WriteFile(path string, data []byte) (err error) {
	file, tmpPath, err := w.open(path)

	// if an error occured make sure that our temporary file is removed
	defer func() {
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	err = w.write(file, data)
	if err != nil {
		return
	}

	// rename file after writing
	err = os.Rename(tmpPath, path)
	return
}

func (w AtomicWriter) CopyFile(path string, src io.Reader) (err error) {
	file, tmpPath, err := w.open(path)

	// if an error occured make sure that our temporary file is removed
	defer func() {
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	err = w.copy(file, src)
	if err != nil {
		return
	}

	// rename file after writing
	err = os.Rename(tmpPath, path)
	return
}
