package log

import (
	"bytes"
	"io"
	"sync"
)

var (
	colorEnd = []byte("\x1b\x5b00m")
	newLine  = []byte{'\n'}
)

type IOWriter struct {
	mutex  sync.Mutex
	out    io.Writer
	level  LLevel
	isatty bool
}

func (w *IOWriter) writeLocked(b []byte) (int, error) {
	n, err := w.out.Write(b)
	if l := len(b); l > 0 && b[l-1] != '\n' && err == nil {
		var n1 int
		n1, err = w.out.Write(newLine)
		n += n1
	}
	return n, err
}

func (w *IOWriter) write(b []byte) (int, error) {
	w.mutex.Lock()
	n, err := w.writeLocked(b)
	w.mutex.Unlock()
	return n, err
}

func (w *IOWriter) writeColored(ll LLevel, colored []byte, uncolored []byte) (n int, err error) {
	w.mutex.Lock()
	var nn int
	nn, err = w.out.Write(ll.colorBeginBytes())
	n += nn
	if err != nil {
		w.mutex.Unlock()
		return
	}
	nn, err = w.out.Write(colored)
	n += nn
	if err != nil {
		w.mutex.Unlock()
		return
	}
	nn, err = w.out.Write(colorEnd)
	n += nn
	if err != nil {
		w.mutex.Unlock()
		return
	}
	nn, err = w.writeLocked(uncolored)
	n += nn
	if err != nil {
		w.mutex.Unlock()
		return
	}
	w.mutex.Unlock()
	return n, nil
}

func (w *IOWriter) Write(level LLevel, flags int, b []byte) (int, error) {
	if w.isatty && flags&(Lshortlevel|Llevel) != 0 && flags&Lcolored != 0 {
		idx := bytes.IndexByte(b, ']')
		if idx > 0 {
			return w.writeColored(level, b[:idx+1], b[idx+1:])
		}
	}
	return w.write(b)
}

func (w *IOWriter) Level() LLevel {
	return w.level
}

func NewIOWriter(out io.Writer, level LLevel) *IOWriter {
	return &IOWriter{out: out, level: level, isatty: isatty(out)}
}
