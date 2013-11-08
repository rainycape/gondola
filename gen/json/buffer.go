// +build IGNORE

package json

// TODO: Make this configurable?
const bufSize = 8192

var (
	buffers = make(chan *bytes.Buffer, runtime.GOMAXPROCS(0))
)

func jsonGetBuffer() *bytes.Buffer {
	var buf *bytes.Buffer
	select {
	case buf = <-buffers:
		buf.Reset()
	default:
		buf = new(bytes.Buffer)
		buf.Grow(bufSize)
	}
	return buf
}

func jsonPutBuffer(buf *bytes.Buffer) {
	if buf.Len() <= bufSize {
		select {
		case buffers <- buf:
		default:
		}
	}
}
