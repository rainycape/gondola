package blobstore

import (
	"bytes"
	"hash"
	"io"

	"gnd.la/blobstore/driver"
)

// WFile represents a file in the blobstore
// opened for writing.
type WFile struct {
	id         string
	file       driver.WFile
	meta       interface{}
	dataHash   hash.Hash64
	dataLength uint64
	store      *Blobstore
	closed     bool
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
	return w.file.Write(p)
}

func (w *WFile) SetMeta(meta interface{}) error {
	w.meta = meta
	return nil
}

// Close closes the file. Once the file is closed, it
// might not be used again.
func (w *WFile) Close() error {
	if !w.closed {
		if err := w.putMeta(); err != nil {
			return err
		}
		return w.file.Close()
	}
	return nil
}

func (w *WFile) putMeta() error {
	if !w.store.drvNoMeta {
		var buf bytes.Buffer
		if err := w.writeMeta(&buf); err != nil {
			return err
		}
		err := w.file.SetMetadata(buf.Bytes())
		if err == driver.ErrMetadataNotHandled {
			// Driver does not support metadata. Update
			// the blobstore and call putMeta() again.
			w.store.drvNoMeta = true
			return w.putMeta()
		}
		return err
	}
	// No meta supported by the driver, generate a .meta
	// file.
	f, err := w.store.drv.Create(w.store.metaName(w.id))
	if err != nil {
		return err
	}
	defer f.Close()
	return w.writeMeta(f)
}

func (w *WFile) writeMeta(out io.Writer) error {
	var err error
	// Write version number
	if err = bwrite(out, uint8(1)); err != nil {
		return err
	}
	// Write flags
	if err = bwrite(out, uint64(0)); err != nil {
		return err
	}
	var metadata []byte
	metadataLength := uint64(0)
	metadataHash := uint64(0)
	if w.meta != nil && !isNil(w.meta) {
		metadata, err = marshal(w.meta)
		if err != nil {
			return err
		}
		metadataLength = uint64(len(metadata))
		h := newHash()
		h.Write(metadata)
		metadataHash = h.Sum64()
	}
	// Metadata metadata
	if err := bwrite(out, metadataLength); err != nil {
		return err
	}
	if err := bwrite(out, metadataHash); err != nil {
		return err
	}
	// Data metadata
	if err := bwrite(out, w.dataLength); err != nil {
		return err
	}
	if err := bwrite(out, w.dataHash.Sum64()); err != nil {
		return err
	}
	if len(metadata) > 0 {
		if _, err := out.Write(metadata); err != nil {
			return err
		}
	}
	return nil
}
