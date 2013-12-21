package pipe

import (
	"bytes"
	"compress/zlib"
	"io/ioutil"
)

// zlib pipe. Compresses data at default level. If
// the data is less than 100 bytes or takes more
// space when compressed, data is returned uncompressed.

func zlibEncode(b []byte) ([]byte, error) {
	if len(b) > 100 {
		var buf bytes.Buffer
		w := zlib.NewWriter(&buf)
		if _, err := w.Write(b); err != nil {
			return nil, err
		}
		if err := w.Close(); err != nil {
			return nil, err
		}
		if buf.Len() < len(b) {
			return buf.Bytes(), nil
		}
	}
	return b, nil
}

func zlibDecode(b []byte) ([]byte, error) {
	// Check for a valid zlib header before calling zlib.NewReader
	if b[0]&0x0f != 8 || len(b) < 2 {
		return b, nil
	}
	h := uint(b[0])<<8 | uint(b[1])
	if h%31 != 0 {
		return b, nil
	}
	if r, err := zlib.NewReader(bytes.NewReader(b)); err == nil {
		if data, err := ioutil.ReadAll(r); err == nil {
			if err := r.Close(); err == nil {
				return data, nil
			}
		}
	}
	return b, nil
}

func init() {
	Register("zlib", &Pipe{
		Encode: zlibEncode,
		Decode: zlibDecode,
	})
}
