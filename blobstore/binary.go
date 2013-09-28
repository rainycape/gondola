package blobstore

import (
	"encoding/binary"
	"io"
)

func bread(r io.Reader, data interface{}) error {
	return binary.Read(r, binary.BigEndian, data)
}

func bwrite(w io.Writer, data interface{}) error {
	return binary.Write(w, binary.BigEndian, data)
}
