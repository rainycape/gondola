package pipe

import (
	"bytes"
	"compress/zlib"
	"io/ioutil"
)

// zlib pipe. Compresses data at default level. If
// the data is less than 100 bytes or takes more
// space when compressed, data is stored uncompressed
// the first byte is used to indicate if the data
// is compressed.

func zlibEncode(b []byte) ([]byte, error) {
	if len(b) > 100 {
		var buf bytes.Buffer
		buf.WriteByte(1)
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
	return append([]byte{0}, b...), nil
}

func zlibDecode(b []byte) ([]byte, error) {
	if b[0] != 0 {
		r, err := zlib.NewReader(bytes.NewReader(b[1:]))
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		if err := r.Close(); err != nil {
			return nil, err
		}
		return data, nil
	}
	return b[1:], nil
}

func init() {
	Register("zlib", &Pipe{
		Encode: zlibEncode,
		Decode: zlibDecode,
	})
}
