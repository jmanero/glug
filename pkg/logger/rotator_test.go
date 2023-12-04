package logger_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmanero/glug/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestRotator(t *testing.T) {
	dir := t.TempDir()

	rotator, err := logger.Open(filepath.Join(dir, "log"), logger.RotatorOptions{
		Enabled:    true,
		MaxSize:    24,
		MinSize:    12,
		MaxAge:     time.Second,
		Count:      2,
		Pattern:    "%Y-%m-%dT%H%M%S.%f",
		CreateMode: 0o644,
	})

	assert.NoError(t, err, "Rotator created without error")

	versions, err := rotator.Versions()
	assert.NoError(t, err, "Globs rotated versions without error")
	assert.Zero(t, versions, "No versions exist")

	// Test MaxSize rotation
	n, err := rotator.Write([]byte("Hello world\n"))
	assert.NoError(t, err, "Write without error")
	assert.Equal(t, 12, n, "Write returns correct byte-length")

	versions, err = rotator.Versions()
	assert.NoError(t, err, "Globs rotated versions without error")
	assert.Empty(t, versions, "No versions exist after first write")

	n, err = rotator.Write([]byte("Hello world\n"))
	assert.NoError(t, err, "Write without error")
	assert.Equal(t, 12, n, "Write returns correct byte-length")

	versions, err = rotator.Versions()
	assert.NoError(t, err, "Globs rotated versions without error")
	assert.Len(t, versions, 1, "One version exists after second write")

	// Test max-age rotation
	rotator.MaxAge = 0
	n, err = rotator.Write([]byte("Hello world\n"))
	assert.NoError(t, err, "Write without error")
	assert.Equal(t, 12, n, "Write returns correct byte-length")

	versions, err = rotator.Versions()
	assert.NoError(t, err, "Globs rotated versions without error")
	assert.Len(t, versions, 2, "Two version exists after third write")

	rotator.MaxAge = time.Second

	// Test Cleanup
	n, err = rotator.Write([]byte("Hello world\n"))
	assert.NoError(t, err, "Write without error")
	assert.Equal(t, 12, n, "Write returns correct byte-length")

	versions, err = rotator.Versions()
	assert.NoError(t, err, "Globs rotated versions without error")
	assert.Len(t, versions, 2, "Two version exists after fourth write")

	n, err = rotator.Write([]byte("Hello world\n"))
	assert.NoError(t, err, "Write without error")
	assert.Equal(t, 12, n, "Write returns correct byte-length")

	// Wait for async cleanup routine
	time.Sleep(500 * time.Millisecond)
	versions, err = rotator.Versions()
	assert.NoError(t, err, "Globs rotated versions without error")
	assert.Len(t, versions, 2, "Two version exists after fifth write")

	// Test count==0/truncation
	rotator.Count = 0

	n, err = rotator.Write([]byte("Hello world\n"))
	assert.NoError(t, err, "Write without error")
	assert.Equal(t, 12, n, "Write returns correct byte-length")

	versions, err = rotator.Versions()
	assert.NoError(t, err, "Globs rotated versions without error")
	assert.Len(t, versions, 2, "Two version exists after sixth write")

	n, err = rotator.Write([]byte("Hello world\n"))
	assert.NoError(t, err, "Write without error")
	assert.Equal(t, 12, n, "Write returns correct byte-length")

	stat, err := os.Stat(rotator.Name())
	assert.NoError(t, err, "Stats output file without error")
	assert.Zero(t, stat.Size(), "Output file is empty after re-opening with TRUNCATE flag")

	// Wait for async cleanup routine
	time.Sleep(500 * time.Millisecond)

	versions, err = rotator.Versions()
	assert.NoError(t, err, "Globs rotated versions without error")
	assert.Empty(t, versions, "Zero version exists for count == 0")
}
