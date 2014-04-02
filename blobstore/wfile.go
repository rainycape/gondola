package blobstore

import (
	"bytes"
	"gnd.la/blobstore/driver"
	"hash"
	"io"
	"os"
)

// WFile represents a file in the blobstore
// opened for writing.
type WFile struct {
	id             string
	metadataLength uint64
	dataLength     uint64
	dataHash       hash.Hash64
	wfile          driver.WFile
	seeker         io.Seeker
	closed         bool
	buf            bytes.Buffer
}

// Id returns the unique file identifier as a string.
func (w *WFile) Id() string {
	return w.id
}

// Write writes the bytes from p into the file. This
// method implements the io.Writer interface.
func (w *WFile) Write(p []byte) (int, error) {
	w.dataHash.Write(p)
	w.dataLength += uint64(len(p))
	if w.seeker != nil {
		return w.wfile.Write(p)
	}
	// The underlying driver does not support seeking, buffer
	// writes.
	return w.buf.Write(p)
}

// Close closes the file. Once the file is closed, it
// might not be used again.
func (w *WFile) Close() error {
	if !w.closed {
		if w.seeker != nil {
			// Seek to the end of the metadata to update the size and hash
			dataLengthPos := int64(1 + 8 + 8 + 8 + w.metadataLength)
			if _, err := w.seeker.Seek(dataLengthPos, os.SEEK_SET); err != nil {
				return err
			}
		}
		if err := bwrite(w.wfile, w.dataLength); err != nil {
			return err
		}
		if err := bwrite(w.wfile, w.dataHash.Sum64()); err != nil {
			return err
		}
		if w.seeker == nil {
			// No seeking, write buffered data
			if _, err := w.wfile.Write(w.buf.Bytes()); err != nil {
				return err
			}
		}
		w.closed = true
		return w.wfile.Close()
	}
	return nil
}
