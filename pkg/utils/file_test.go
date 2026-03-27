package utils

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestCreate(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)

	path := filepath.Join(baseDir, "../sstable/records/manifest.txt")

	f := NewFile("manifest.txt", path)
	err := f.Create()
	if err != nil {
		return
	}
}
