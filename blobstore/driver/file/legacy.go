package file

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"gnd.la/blobstore/driver"
)

func bread(r io.Reader, data interface{}) error {
	return binary.Read(r, binary.BigEndian, data)
}

type legacyFile struct {
	*bytes.Reader
	meta         []byte
	dataHash     uint64
	metadataHash uint64
}

func (f *legacyFile) Close() error {
	return nil
}

func (f *legacyFile) Metadata() ([]byte, error) {
	return nil, driver.ErrMetadataNotHandled
}

func readLegacyFile(r *os.File) (*legacyFile, error) {
	var version uint8
	var err error
	if err = bread(r, &version); err != nil {
		return nil, err
	}
	if version != 1 {
		return nil, fmt.Errorf("can't read files with version %d", version)
	}
	// Skip over the flags for now
	var flags uint64
	if err = bread(r, &flags); err != nil {
		return nil, err
	}
	var metadataLength uint64
	if err = bread(r, &metadataLength); err != nil {
		return nil, err
	}
	var file legacyFile
	if err = bread(r, &file.metadataHash); err != nil {
		return nil, err
	}
	if metadataLength > 0 {
		file.meta = make([]byte, int(metadataLength))
		if _, err = io.ReadFull(r, file.meta); err != nil {
			return nil, err
		}
	}
	var dataLength uint64
	if err = bread(r, &dataLength); err != nil {
		return nil, err
	}
	if err = bread(r, &file.dataHash); err != nil {
		return nil, err
	}
	// The rest is actual contents
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, err
	}
	if uint64(buf.Len()) != dataLength {
		return nil, fmt.Errorf("len mismatch %d vs %d", buf.Len(), dataLength)
	}
	file.Reader = bytes.NewReader(buf.Bytes())
	return &file, nil
}
