package upload

import (
	"os"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func exists(t *testing.T, path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		assert.NilError(t, err)
	}
	return true
}

func TestCleanup(t *testing.T) {
	t.Parallel()
	r := NewFileRegistry(FileRegistryConfig{ExpireInterval: time.Millisecond * 100})
	defer r.Stop()

	// create file
	dir := t.TempDir()
	path := dir + "/test"
	_, err := os.Create(path)
	assert.NilError(t, err)

	// register timeout
	r.RegisterFile(path, time.Now().Add(time.Millisecond*100))

	// should exist still
	assert.Assert(t, exists(t, path))

	time.Sleep(time.Millisecond * 200)

	// should be gone
	assert.Assert(t, !exists(t, path))
}

func TestKeep(t *testing.T) {
	t.Parallel()
	r := NewFileRegistry(FileRegistryConfig{ExpireInterval: time.Millisecond * 50, KeepDelay: time.Millisecond})
	defer r.Stop()

	ch := make(chan struct{})

	// create file
	dir := t.TempDir()
	path := dir + "/test"
	_, err := os.Create(path)
	assert.NilError(t, err)

	// register timeout
	r.KeepFile(path, ch)

	// should exist still
	assert.Assert(t, exists(t, path))

	// close channel
	close(ch)
	time.Sleep(time.Millisecond * 200)

	// should be gone
	assert.Assert(t, !exists(t, path))
}

// TestKeepPriority tests that a keep channel is not dropped when a new
// registration is made
func TestKeepPriority(t *testing.T) {
	t.Parallel()
	r := NewFileRegistry(FileRegistryConfig{ExpireInterval: time.Millisecond * 50, KeepDelay: time.Millisecond})
	defer r.Stop()

	ch := make(chan struct{})
	defer close(ch)

	// create file
	dir := t.TempDir()
	path := dir + "/test"
	_, err := os.Create(path)
	assert.NilError(t, err)

	// register timeout
	r.KeepFile(path, ch)

	// override with timeout
	r.RegisterFile(path, time.Now().Add(time.Millisecond*100))

	time.Sleep(time.Millisecond * 200)

	// should exist still
	assert.Assert(t, exists(t, path))
}
