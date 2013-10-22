package util

import (
	"fmt"
	"io"
	"os"
)

// WriteFile writes the given data to the file named by filename. If the file already
// exists and overwrite is false, it will return an error. If filename is empty or
// the string "-" it will write to os.Stdout.
func WriteFile(filename string, data []byte, overwrite bool, perm os.FileMode) error {
	var w io.Writer
	if filename != "" && filename != "-" {
		flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		if !overwrite {
			flags |= os.O_EXCL
		}
		f, err := os.OpenFile(filename, flags, perm)
		if err != nil {
			return fmt.Errorf("error creating output file %s: %s\n", filename, err)
		}
		defer f.Close()
		w = f
	} else {
		w = os.Stdout
	}
	_, err := w.Write(data)
	return err
}
