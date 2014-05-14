package leveldb

import (
	"fmt"
	"io"
	"os"
)

type rfile struct {
	metadata []byte
	chunks   [][]byte
	chunk    int
	pos      int
}

func (f *rfile) Metadata() ([]byte, error) {
	return f.metadata, nil
}

func (f *rfile) Seek(offset int64, whence int) (int64, error) {
	var chunk, pos int
	switch whence {
	case os.SEEK_SET:
		chunk = 0
		pos = 0
	case os.SEEK_CUR:
		chunk = f.chunk
		pos = f.pos
	case os.SEEK_END:
		chunk = len(f.chunks) - 1
		pos = len(f.chunks[chunk])
	default:
		return 0, fmt.Errorf("invalid whence %d", whence)
	}
	res := pos + int(offset)
	for res < 0 {
		if chunk < 0 {
			return 0, fmt.Errorf("can't seek to negative offset %d", res)
		}
		res += len(f.chunks[chunk])
		chunk--
	}
	for res >= len(f.chunks[chunk]) {
		if chunk >= len(f.chunks) {
			res = 0
			break
		}
		res -= len(f.chunks[chunk])
		chunk++
	}
	f.pos = res
	f.chunk = chunk
	for ii := 0; ii < chunk; ii++ {
		res += len(f.chunks[ii])
	}
	return int64(res), nil
}

func (f *rfile) Read(p []byte) (int, error) {
	total := len(p)
	n := 0
	for {
		if f.chunk >= len(f.chunks) {
			return n, io.EOF
		}
		chunk := f.chunks[f.chunk]
		nn := copy(p[n:], chunk[f.pos:])
		n += nn
		f.pos += nn
		if f.pos == len(chunk) {
			f.chunk++
			f.pos = 0
		}
		if n == total {
			return n, nil
		}
	}
	panic("unreachable")
}

func (f *rfile) Close() error {
	return nil
}
