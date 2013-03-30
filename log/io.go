package log

import (
	"bytes"
	"io"
	"sync"
)

type IOWriter struct {
	mutex  sync.Mutex
	out    io.Writer
	level  LLevel
	isatty bool
}

func (w *IOWriter) Write(level LLevel, flags int, b []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	var n int
	var err error
	if w.isatty && flags&(Lshortlevel|Llevel) != 0 && flags&Lcolored != 0 {
		idx := bytes.Index(b, []byte{']'})
		if idx > 0 {
			var n1 int
			n, err = w.out.Write([]byte("\x1b\x5b" + level.Colorcode() + "m"))
			if err != nil {
				return n, err
			}
			idx++
			n1, err = w.out.Write(b[:idx])
			n += n1
			if err != nil {
				return n, err
			}
			n1, err = w.out.Write([]byte("\x1b\x5b00m"))
			n += n1
			if err != nil {
				return n, err
			}
			n1, err = w.out.Write(b[idx:])
			n += n1
			if err != nil {
				return n, err
			}
		}
	} else {
		n, err = w.out.Write(b)
	}
	if l := len(b); l > 0 && b[l-1] != '\n' && err == nil {
		var n1 int
		n1, err = w.out.Write([]byte{'\n'})
		n += n1
	}
	return n, err
}

func (w *IOWriter) Level() LLevel {
	return w.level
}

func NewIOWriter(out io.Writer, level LLevel) *IOWriter {
	return &IOWriter{out: out, level: level, isatty: isatty(out)}
}
