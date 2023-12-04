package logger

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/itchyny/timefmt-go"
	"go.uber.org/multierr"
	"storj.io/common/memory"
)

// RotatorOptions for log rotation
type RotatorOptions struct {
	Enabled bool

	MaxSize memory.Size
	MinSize memory.Size
	MaxAge  time.Duration
	Count   int

	Pattern    string
	CreateMode FileMode
}

// Run pipes log lines from a reader to a file at the given path.
func Run(ctx context.Context, src io.Reader, path string, opts RotatorOptions) (err error) {
	rotator, err := Open(path, opts)
	if err != nil {
		return
	}

	defer rotator.Close()
	return rotator.Pipe(ctx, src)
}

// Rotate performs a one-shot rotation on the given log path
func Rotate(_ context.Context, path string, opts RotatorOptions) (_ bool, err error) {
	rotator, err := Open(path, opts)
	if err != nil {
		return
	}

	defer rotator.Close()
	return rotator.Rotate()
}

// WriteRotator extends io.WriteRotator with methods to aide in log-file rotation
type WriteRotator interface {
	io.WriteCloser
	sync.Locker

	Open(string, fs.FileMode) error
	Reopen(string, fs.FileMode) error
	Truncate() error

	Age() time.Duration
	Name() string
	Created() time.Time
	Size() int64
}

// Rotator provides an io.WriteCloser that applies a rotation policy to its output file
type Rotator struct {
	RotatorOptions
	WriteRotator
}

// Open configures a new Rotator and loads the current state of the output file
func Open(name string, opts RotatorOptions) (_ *Rotator, err error) {
	rotator := &Rotator{RotatorOptions: opts}

	rotator.WriteRotator, err = OpenFileWriter(name, rotator.Mode())
	if err != nil {
		return nil, err
	}

	return rotator, nil
}

// Mode returns the writer's configured CreateMode
func (rotator *Rotator) Mode() fs.FileMode {
	return fs.FileMode(rotator.CreateMode)
}

// Versions lists rotated files in the same directory as the current output file
func (rotator *Rotator) Versions() (versions []string, err error) {
	versions, err = filepath.Glob(rotator.Name() + ".*")
	if err != nil {
		return
	}

	// Sort by timestamp-suffix
	slices.Sort(versions)

	return
}

// NeedsRotation checks if the output file needs to be rotated
func (rotator *Rotator) NeedsRotation() bool {
	if !rotator.Enabled {
		return false
	}

	// Get a single value for the current output file size. Access is not synchronized
	size := rotator.Size()

	// Rotate on output file size
	if size >= int64(rotator.MaxSize) {
		return true
	}

	// Rotate on output file age if size meets the minimum threshold
	if rotator.Age() > rotator.MaxAge && size >= int64(rotator.MinSize) {
		return true
	}

	return false
}

// Rotate closes, renames, then reopens the output file if it requires rotation according to the Writer's configuration
func (rotator *Rotator) Rotate() (rotated bool, err error) {
	if !rotator.NeedsRotation() {
		return
	}

	if rotator.Count == 0 {
		// Special case: Truncate the output file in place
		err = rotator.Truncate()
	} else {
		// Rename the current file with a timestamp suffix then create a new empty output file
		err = rotator.Reopen(rotator.Name()+"."+timefmt.Format(time.Now().UTC(), rotator.Pattern), rotator.Mode())
	}

	if err != nil {
		return
	}

	// Run version cleanup asynchronously
	go rotator.Cleanup()
	return true, nil
}

// Cleanup attempts to remove outdated rotated files
func (rotator *Rotator) Cleanup() (err error) {
	// Count == -1 disables cleanup
	if rotator.Count < 0 {
		return
	}

	// Find timestamp-suffixed output file versions
	versions, err := rotator.Versions()
	if err != nil {
		return
	}

	if remove := len(versions) - rotator.Count; remove > 0 {
		// Remove oldest (first in sorted slice) rotated files, retaining newest $Count files
		versions = versions[:remove]

		for _, version := range versions {
			// Try to remove all outdated versions
			err = multierr.Append(err, os.Remove(version))
		}
	}

	return
}

// Pipe reads from a source io.Reader to the Writer's rotated output file
func (rotator *Rotator) Pipe(ctx context.Context, src io.Reader) (err error) {
	_, err = io.Copy(rotator, NewCancelReader(ctx, src))
	return
}

// Write to the output file then check if rotation is required
func (rotator *Rotator) Write(chunk []byte) (n int, err error) {
	n, err = rotator.WriteRotator.Write(chunk)
	if err != nil {
		return
	}

	// Check if rotation is required
	_, err = rotator.Rotate()
	return
}
