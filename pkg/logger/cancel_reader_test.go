package logger_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/jmanero/glug/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func NewCancelReaderPipe(ctx context.Context) (io.ReadCloser, *io.PipeWriter) {
	reader, writer := io.Pipe()
	return logger.NewCancelReader(ctx, reader), writer
}

func TestReadPreemption(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader, _ := NewCancelReaderPipe(ctx)

	t.Run("Unblocks read on cancellation", func(t *testing.T) {
		t.Parallel()

		_, err := reader.Read(nil)
		assert.ErrorIs(t, err, io.EOF, "Returns EOF after cancellation")
	})

	t.Run("Cancel read", func(t *testing.T) {
		t.Parallel()
		cancel()
	})
}

func TestReadAfterClose(t *testing.T) {
	reader, _ := NewCancelReaderPipe(context.Background())
	reader.Close()

	_, err := reader.Read(nil)
	assert.ErrorIs(t, err, io.ErrClosedPipe, "Returns error ErrClosedPipe after close")
}

func TestReadError(t *testing.T) {
	reader, writer := NewCancelReaderPipe(context.Background())
	writer.CloseWithError(io.ErrUnexpectedEOF)

	_, err := reader.Read(nil)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF, "Returns error from wrapped reader")
}

func TestReadTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	reader, _ := NewCancelReaderPipe(ctx)

	_, err := reader.Read(nil)
	assert.ErrorIs(t, err, io.ErrNoProgress, "Returns ErrNoProgress after timeout")

	cancel()
}

func TestCancelAfterWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader, writer := NewCancelReaderPipe(ctx)

	writer.Write([]byte("Hello World"))

	// Give the worker routine some time to read from the pipe and write to the
	// internal buffer channel before closing it in the watcher routine
	time.Sleep(time.Second)
	cancel()

	buf := make([]byte, 1024)
	n, err := reader.Read(buf)

	assert.NoError(t, err, "Does not return EOF if buffer contains a chunk")
	assert.Equal(t, 11, n)
	assert.Equal(t, []byte("Hello World"), buf[:n])

	_, err = reader.Read(nil)
	assert.ErrorIs(t, err, io.EOF, "Returns EOF after cancellation")
}

func TestShortBufferCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader, writer := NewCancelReaderPipe(ctx)

	writer.Write([]byte("Hello World"))

	_, err := reader.Read(make([]byte, 4))
	assert.ErrorIs(t, err, io.ErrShortBuffer, "Returns ErrShortBuffer for a small read buffer")

	cancel()

	_, err = reader.Read(nil)
	assert.ErrorIs(t, err, io.EOF, "Returns EOF after cancellation")
}
