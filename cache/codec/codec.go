package codec

var (
	codecs = map[string]*Codec{}
)

type Codec struct {
	Encode func(v interface{}) ([]byte, error)
	Decode func(data []byte, v interface{}) error
}

func Register(name string, c *Codec) {
	codecs[name] = c
}

func Get(name string) *Codec {
	return codecs[name]
}
