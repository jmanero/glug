package logger

import (
	"io/fs"
	"strconv"
)

// FileMode extends fs.FileMode with the pflag.Value interface
type FileMode fs.FileMode

// Set value from a string argument
func (mode *FileMode) Set(value string) (err error) {
	parsed, err := strconv.ParseUint(value, 0, 32)
	if err != nil {
		return
	}

	*mode = FileMode(parsed)
	return
}

func (mode FileMode) String() string {
	return "0" + strconv.FormatUint(uint64(mode), 8)
}

// Type description for CLI usage
func (FileMode) Type() string {
	return "fs.FileMode"
}
