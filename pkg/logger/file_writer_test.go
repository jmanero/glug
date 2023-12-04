package logger_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmanero/glug/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func NewFileWriterBench(t *testing.T) (dir string, writer *logger.FileWriter) {
	dir = t.TempDir()

	writer, err := logger.OpenFileWriter(filepath.Join(dir, "log"), 0o644)
	assert.NoError(t, err, "Opens output file")

	n, err := writer.Write([]byte("Hello World\n"))
	assert.NoError(t, err, "Writers to output file")
	assert.Equal(t, 12, n)
	assert.Equal(t, int64(12), writer.Size())

	data, err := os.ReadFile(filepath.Join(dir, "log"))
	assert.NoError(t, err, "Test reads back output file")
	assert.Equal(t, 12, len(data))
	assert.Equal(t, []byte("Hello World\n"), data)

	return
}

func TestFileWrite(t *testing.T) {
	dir := t.TempDir()

	writer, err := logger.OpenFileWriter(filepath.Join(dir, "log"), 0o644)
	assert.NoError(t, err, "Opens output file")

	n, err := writer.Write([]byte("Hello"))
	assert.NoError(t, err, "Writers to output file")
	assert.Equal(t, 5, n)
	assert.Equal(t, int64(5), writer.Size())

	n, err = writer.Write([]byte(" World\n"))
	assert.NoError(t, err, "Writers to output file again")
	assert.Equal(t, 7, n)
	assert.Equal(t, int64(12), writer.Size())

	assert.NoError(t, writer.Close(), "Syncs and closes output file")

	data, err := os.ReadFile(filepath.Join(dir, "log"))
	assert.NoError(t, err, "Test reads back output file")
	assert.Equal(t, 12, len(data))
	assert.Equal(t, []byte("Hello World\n"), data)
}

func TestFileTruncate(t *testing.T) {
	dir, writer := NewFileWriterBench(t)

	err := writer.Truncate()
	assert.NoError(t, err, "Truncates output file")

	data, err := os.ReadFile(filepath.Join(dir, "log"))
	assert.NoError(t, err, "Test reads back truncated file")
	assert.Equal(t, 0, len(data))
	assert.Empty(t, data, "File is empty")
}

func TestFileReopen(t *testing.T) {
	dir, writer := NewFileWriterBench(t)

	err := writer.Reopen(filepath.Join(dir, "log.rotated"), 0o644)
	assert.NoError(t, err, "Rotates output file")

	data, err := os.ReadFile(filepath.Join(dir, "log.rotated"))
	assert.NoError(t, err, "Test reads back output file")
	assert.Equal(t, 12, len(data))
	assert.Equal(t, []byte("Hello World\n"), data)

	data, err = os.ReadFile(filepath.Join(dir, "log"))
	assert.NoError(t, err, "Test reads back truncated file")
	assert.Equal(t, 0, len(data))
	assert.Empty(t, data, "File is empty")
}

func TestFileAppend(t *testing.T) {
	dir, writer0 := NewFileWriterBench(t)

	assert.NoError(t, writer0.Close(), "Syncs and closes output file")

	writer, err := logger.OpenFileWriter(filepath.Join(dir, "log"), 0o644)
	assert.NoError(t, err, "Reopens output file for appending")
	assert.Equal(t, int64(12), writer.Size(), "Loads correct size from existing file")

	// When a new file is created, writer.Created is set from time.Now(), which should be close to the ctime of the actual new file
	assert.WithinDuration(t, writer0.Created(), writer.Created(), time.Millisecond, "Loads correct ctime from existing file")

	n, err := writer.Write([]byte("Hello World\n"))
	assert.NoError(t, err, "Appends to output file")
	assert.Equal(t, 12, n)
	assert.Equal(t, int64(24), writer.Size())

	assert.NoError(t, writer.Close(), "Syncs and closes output file")

	data, err := os.ReadFile(filepath.Join(dir, "log"))
	assert.NoError(t, err, "Test reads back output file")
	assert.Equal(t, 24, len(data))
	assert.Equal(t, []byte("Hello World\nHello World\n"), data)
}
