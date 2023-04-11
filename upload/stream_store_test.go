package upload

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestUpdateStream(t *testing.T) {
	t.Parallel()
	s := NewStreamStore(StreamStoreConfig{
		StreamTimeout:        time.Millisecond * 100,
		StreamExpireInterval: time.Millisecond * 50,
	})
	defer s.Stop()

	// should be nil
	assert.Assert(t, s.GetStream("test") == nil)

	assert.NilError(t, s.UpdateStream("test", "foo.com"))
	assert.Assert(t, s.GetStream("test") != nil)
	time.Sleep(200 * time.Millisecond)
	assert.Assert(t, s.GetStream("test") == nil)
}

func TestPreventMultipleOrigin(t *testing.T) {
	t.Parallel()
	s := NewStreamStore(StreamStoreConfig{
		StreamTimeout:        time.Millisecond * 200,
		StreamExpireInterval: time.Millisecond * 50,
	})
	defer s.Stop()

	// prevent multiple parallel origins
	assert.NilError(t, s.UpdateStream("test", "foo.com"))
	assert.ErrorIs(t, s.UpdateStream("test", "bar.com"), errInvalidOrigin)

	time.Sleep(150 * time.Millisecond)

	// allow after sufficient time has passed
	assert.NilError(t, s.UpdateStream("test", "bar.com"))
}
