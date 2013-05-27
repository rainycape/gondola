package driver

var (
	drivers = map[string]Opener{}
)

type Options map[string]string

func (o Options) Get(key string) string {
	return o[key]
}

type Opener func(value string, o Options) Driver

type Driver interface {
	Set(key string, b []byte, timeout int) error
	Get(key string) ([]byte, error)
	GetMulti(keys []string) (map[string][]byte, error)
	Delete(key string) error
	Close() error
	Connection() interface{}
}

func Register(name string, f Opener) {
	drivers[name] = f
}

func Get(name string) Opener {
	return drivers[name]
}
