package logger

import (
	"context"
	"errors"
	"io"
)

// CancelReader implements a preempt-able io.Reader based upon https://benjamincongdon.me/blog/2020/04/23/Cancelable-Reads-in-Go/
type CancelReader struct {
	stop context.CancelCauseFunc
	data chan []byte
	err  error
}

// NewCancelReader certes a preempt-able io.ReadCloser that wraps an io.Reader
func NewCancelReader(ctx context.Context, src io.Reader) io.ReadCloser {
	reader := &CancelReader{data: make(chan []byte)}
	ctx, reader.stop = context.WithCancelCause(ctx)

	go reader.worker(ctx, src)
	go reader.watcher(ctx)

	return reader
}

// Read from the internal buffer
func (reader *CancelReader) Read(buf []byte) (n int, _ error) {
	for {
		chunk, ok := <-reader.data
		if !ok {
			return 0, reader.err
		}

		n = copy(buf, chunk)
		if n < len(chunk) {
			return n, io.ErrShortBuffer
		}

		return n, nil
	}
}

// Close causes Read to return ErrClosedPipe on the next call after the internal buffer has been drained
func (reader *CancelReader) Close() error {
	reader.stop(io.ErrClosedPipe)
	return nil
}

// worker reads from the source Reader to the internal buffer
func (reader *CancelReader) worker(ctx context.Context, src io.Reader) {
	buffer := make([]byte, 1024)

	// Squash a panic from writing to a closed channel if the reader was preempted during a blocking read
	defer func() { recover() }()

	for {
		// Check for context cancellation before starting a blocking read
		if ctx.Err() != nil {
			return
		}

		n, err := src.Read(buffer)
		if n > 0 {
			reader.data <- buffer[:n]
		}

		if err != nil {
			// Pass a read-error or EOF to the watcher routine
			reader.stop(err)
			return
		}
	}
}

// watcher closes the buffer channel after the reader's context is canceled
func (reader *CancelReader) watcher(ctx context.Context) {
	<-ctx.Done()
	err := context.Cause(ctx)

	// Alias generic context errors to generic io errors
	if errors.Is(err, context.Canceled) {
		reader.err = io.EOF
	} else if errors.Is(err, context.DeadlineExceeded) {
		reader.err = io.ErrNoProgress
	} else {
		reader.err = err
	}

	// Stop blocking reader.Read, cause worker goroutine to join via panic/recover
	// after next read if blocked
	close(reader.data)
	return
}
