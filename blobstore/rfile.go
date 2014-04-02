package blobstore

import (
	"errors"
	"gnd.la/blobstore/driver"
	"io"
	"io/ioutil"
	"os"
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
	metadataData []byte
	metadataHash uint64
	dataLength   uint64
	dataHash     uint64
	rfile        driver.RFile
}

// Id returns the unique file identifier as a string.
func (r *RFile) Id() string {
	return r.id
}

// Read reads from the file into the p buffer. This
// method implements the io.Reader interface.
func (r *RFile) Read(p []byte) (int, error) {
	return r.rfile.Read(p)
}

// Close closes the file. Once the file is closed, it
// might not be used again.
func (r *RFile) Close() error {
	return r.rfile.Close()
}

// ReadAll is a shorthand for ioutil.ReadAll(r)
func (r *RFile) ReadAll() ([]byte, error) {
	return ioutil.ReadAll(r)
}

// Seek implements the same semantics than os.File.Seek.
func (r *RFile) Seek(offset int64, whence int) (int64, error) {
	// Version + flags + metadata size + metadata hash + metadata length + data size + data hash
	dataStart := int64(1 + 8 + 8 + 8 + len(r.metadataData) + 8 + 8)
	switch whence {
	case os.SEEK_SET:
		offset += dataStart
		pos, err := r.rfile.Seek(offset, whence)
		if err == nil {
			pos -= dataStart
		}
		return pos, err
	case os.SEEK_CUR, os.SEEK_END:
		pos, err := r.rfile.Seek(offset, whence)
		if err == nil {
			if pos < dataStart {
				return r.Seek(0, os.SEEK_SET)
			}
			pos -= dataStart
		}
		return pos, err
	}
	// User passed something other than -1, 0 and 1.
	panic("invalid whence")
}

// GetMeta retrieves the file metadata, previously stored
// when writing the file, into the meta argument, which
// must be a pointer.
func (r *RFile) GetMeta(meta interface{}) error {
	if r.metadataData != nil {
		return unmarshal(r.metadataData, meta)
	}
	return nil
}

// Verify checks the integrity of both the data and
// the metadata in the file.
func (r *RFile) Verify() error {
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

// Size returns the size of the file (without the metadata or
// any addtional data added by the storage system).
func (r *RFile) Size() uint64 {
	return r.dataLength
}
