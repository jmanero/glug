package logger

import (
	"errors"
	"io/fs"
	"os"
	"sync"
	"syscall"
	"time"

	"go.uber.org/multierr"
)

// FileWriter provides a thread-safe io.WriteCloser and helper methods for writing and rotating an output file
type FileWriter struct {
	sync.RWMutex

	file    *os.File
	size    int64
	created time.Time
}

// Assert that FileWriter implements WriteRotator
var _ WriteRotator = &FileWriter{}

// OpenFileWriter creates and initializes a new FileWriter at the given path
func OpenFileWriter(name string, mode fs.FileMode) (*FileWriter, error) {
	writer := new(FileWriter)
	if err := writer.Open(name, mode); err != nil {
		return nil, err
	}

	return writer, nil
}

// Write to the current output file
func (writer *FileWriter) Write(buf []byte) (n int, err error) {
	writer.Lock()
	defer writer.Unlock()

	n, err = writer.file.Write(buf)

	// Count bytes written while we have an exclusive lock
	writer.size += int64(n)

	return
}

func (writer *FileWriter) create(name string, mode fs.FileMode) (err error) {
	writer.size = 0
	writer.created = time.Now().UTC()

	writer.file, err = os.OpenFile(name, os.O_WRONLY|os.O_CREATE, mode)
	return
}

// Open tries to create or append to an output file
func (writer *FileWriter) Open(name string, mode fs.FileMode) (err error) {
	stat, err := os.Stat(name)
	if errors.Is(err, os.ErrNotExist) {
		// Open a new empty file
		err = writer.create(name, mode)

		return
	}

	if err != nil {
		// Return unhandled error from Stat
		return
	}

	// Get file ctime: linux and darwin both implement this type
	sys := stat.Sys().(*syscall.Stat_t)

	// Open an existing file for appending
	writer.size = stat.Size()
	writer.created = time.Unix(sys.Ctimespec.Sec, sys.Ctimespec.Nsec)

	writer.file, err = os.OpenFile(name, os.O_WRONLY|os.O_APPEND, 0)
	return
}

// Reopen rotates the output file by closing the existing handle, renaming the file, then creating a new file at the same path
func (writer *FileWriter) Reopen(rename string, mode fs.FileMode) (err error) {
	writer.Lock()
	defer writer.Unlock()

	err = writer.file.Sync()
	if err != nil {
		return
	}

	err = writer.file.Close()
	if err != nil {
		return
	}

	err = os.Rename(writer.file.Name(), rename)
	if err != nil {
		return
	}

	return writer.create(writer.file.Name(), mode)
}

// Truncate closes and reopens the output file with the TRUNCATE flag set
func (writer *FileWriter) Truncate() (err error) {
	writer.Lock()
	defer writer.Unlock()

	err = writer.file.Sync()
	if err != nil {
		return
	}

	err = writer.file.Close()
	if err != nil {
		return
	}

	writer.file, err = os.OpenFile(writer.file.Name(), os.O_WRONLY|os.O_TRUNC, 0)
	return
}

// Close the underlying file
func (writer *FileWriter) Close() (err error) {
	err = multierr.Append(err, writer.file.Sync())
	err = multierr.Append(err, writer.file.Close())

	return
}

// Age is a helper to calculate the current duration since the writer's cached ctime
func (writer *FileWriter) Age() time.Duration {
	return time.Since(writer.Created())
}

// Created is a synchronized getter for the writer's cached ctime
func (writer *FileWriter) Created() time.Time {
	writer.RLock()
	defer writer.RUnlock()
	return writer.created
}

// Name returns the name of the file as presented to Open
func (writer *FileWriter) Name() string {
	return writer.file.Name()
}

// Size is a synchronized getter for the current cached file-size value
func (writer *FileWriter) Size() int64 {
	writer.RLock()
	defer writer.RUnlock()
	return writer.size
}
