package blobstore

import (
	"gnd.la/blobstore/driver"
	"hash"
	"os"
)

type WFile struct {
	id             string
	metadataLength uint64
	dataLength     uint64
	dataHash       hash.Hash64
	wfile          driver.WFile
	closed         bool
}

func (w *WFile) Id() string {
	return w.id
}

func (w *WFile) Write(p []byte) (int, error) {
	w.dataHash.Write(p)
	w.dataLength += uint64(len(p))
	return w.wfile.Write(p)
}

func (w *WFile) Close() error {
	if !w.closed {
		// Seek to the end of the metadata to update
		// data size and data hash. Go back the length
		// of the data as well as 16 bytes.
		if _, err := w.wfile.Seek(-int64(w.dataLength+16), os.SEEK_CUR); err != nil {
			return err
		}
		if err := bwrite(w.wfile, w.dataLength); err != nil {
			return err
		}
		if err := bwrite(w.wfile, w.dataHash.Sum64()); err != nil {
			return err
		}
		w.closed = true
		return w.wfile.Close()
	}
	return nil
}
