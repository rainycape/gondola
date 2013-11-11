// +build IGNORE

package json

var (
	jsonBuffers = make(chan *bytes.Buffer, jsonBufferCount)
)

func jsonGetBuffer() *bytes.Buffer {
	var buf *bytes.Buffer
	select {
	case buf = <-jsonBuffers:
		buf.Reset()
	default:
		buf = new(bytes.Buffer)
		buf.Grow(jsonBufSize)
	}
	return buf
}

func jsonPutBuffer(buf *bytes.Buffer) {
	if buf.Len() <= jsonMaxBufSize {
		select {
		case jsonBuffers <- buf:
		default:
		}
	}
}
