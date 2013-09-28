package file

import (
	"os"
	"path/filepath"
)

type wfile struct {
	*os.File
	path string
}

func (f *wfile) Close() error {
	// Close the file
	if err := f.File.Close(); err != nil {
		return err
	}
	// Create dirs if needed
	if err := os.MkdirAll(filepath.Dir(f.path), 0755); err != nil {
		return err
	}
	// Move the file to its final destination
	if err := os.Rename(f.Name(), f.path); err != nil {
		return err
	}
	return nil
}
