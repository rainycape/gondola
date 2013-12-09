package unidecode

import (
	"compress/zlib"
	"gnd.la/encoding/binary"
	"io"
	"strings"
	"sync"
)

var (
	decoded          = false
	mutex            sync.Mutex
	transliterations [65536][]rune
)

func decodeTransliterations() {
	r, err := zlib.NewReader(strings.NewReader(tableData))
	if err != nil {
		panic(err)
	}
	defer r.Close()
	for {
		var chr uint16
		err := binary.Read(r, binary.LittleEndian, &chr)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		var sl uint8
		err = binary.Read(r, binary.LittleEndian, &sl)
		b := make([]byte, int(sl))
		if _, err := r.Read(b); err != nil {
			panic(err)
		}
		transliterations[int(chr)] = []rune(string(b))
	}
}
