package blobstore

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"gnd.la/blobstore/driver"
)

var (
	// ErrInvalidMetadataHash indicates that the metadata hash
	// does not match the expected value and the file is likely
	// to be corrupted.
	ErrInvalidMetadataHash = errors.New("the metadata hash is invalid")
	// ErrInvalidDataHash indicates that the data hash
	// does not match the expected value and the file is likely
	// to be corrupted.
	ErrInvalidDataHash = errors.New("the data hash is invalid")
)

// RFile represents a blobstore file opened
// for reading.
type RFile struct {
	id           string
	file         driver.RFile
	store        *Blobstore
	hasMeta      bool
	metadataData []byte
	metadataHash uint64
	dataLength   uint64
	dataHash     uint64
}

// Id returns the unique file identifier as a string.
func (r *RFile) Id() string {
	return r.id
}

// Read reads from the file into the p buffer. This
// method implements the io.Reader interface.
func (r *RFile) Read(p []byte) (int, error) {
	return r.file.Read(p)
}

// Close closes the file. Once the file is closed, it
// might not be used again.
func (r *RFile) Close() error {
	return r.file.Close()
}

// ReadAll is a shorthand for ioutil.ReadAll(r)
func (r *RFile) ReadAll() ([]byte, error) {
	return ioutil.ReadAll(r)
}

// Seek implements the same semantics than os.File.Seek.
func (r *RFile) Seek(offset int64, whence int) (int64, error) {
	return r.file.Seek(offset, whence)
}

// GetMeta retrieves the file metadata, previously stored
// when writing the file, into the meta argument, which
// must be a pointer.
func (r *RFile) GetMeta(meta interface{}) error {
	if err := r.decodeMeta(); err != nil {
		return err
	}
	if r.metadataData != nil {
		return unmarshal(r.metadataData, meta)
	}
	return nil
}

// Check checks the integrity of both the data and
// the metadata in the file. If this function returns
// a non-nil error, the file should be considered
// corrupted.
func (r *RFile) Check() error {
	if err := r.decodeMeta(); err != nil {
		return err
	}
	if r.metadataHash != 0 {
		mh := newHash()
		mh.Write(r.metadataData)
		if mh.Sum64() != r.metadataHash {
			return ErrInvalidMetadataHash
		}
	}
	pos, err := r.Seek(0, os.SEEK_CUR)
	if err != nil {
		return err
	}
	defer r.Seek(pos, os.SEEK_SET)
	_, err = r.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}
	dh := newHash()
	_, err = io.Copy(dh, r)
	if err != nil {
		return err
	}
	if dh.Sum64() != r.dataHash {
		return ErrInvalidDataHash
	}
	return nil
}

// Size returns the size of the file stored file.
func (r *RFile) Size() (uint64, error) {
	if err := r.decodeMeta(); err != nil {
		return 0, err
	}
	return r.dataLength, nil
}

func (r *RFile) decodeMeta() error {
	if !r.hasMeta {
		f, err := r.store.drv.Open(r.store.metaName(r.id))
		if err != nil {
			return err
		}
		defer f.Close()
		var version uint8
		if err = bread(f, &version); err != nil {
			return err
		}
		if version != 1 {
			return fmt.Errorf("can't read metadata files with version %d", version)
		}
		// Skip over the flags for now
		var flags uint64
		if err = bread(f, &flags); err != nil {
			return err
		}
		var metadataLength uint64
		if err = bread(f, &metadataLength); err != nil {
			return err
		}
		if err = bread(f, &r.metadataHash); err != nil {
			return err
		}
		if err = bread(f, &r.dataLength); err != nil {
			return err
		}
		if err = bread(f, &r.dataHash); err != nil {
			return err
		}
		if metadataLength > 0 {
			r.metadataData = make([]byte, int(metadataLength))
			if _, err = io.ReadFull(f, r.metadataData); err != nil {
				return err
			}
		}
		r.hasMeta = true
	}
	return nil
}
