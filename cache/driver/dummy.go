package driver

type DummyDriver struct {
}

func (c *DummyDriver) Set(key string, b []byte, timeout int) error {
	return nil
}

func (c *DummyDriver) Get(key string) ([]byte, error) {
	return nil, nil
}

func (c *DummyDriver) GetMulti(keys []string) (map[string][]byte, error) {
	return nil, nil
}

func (c *DummyDriver) Delete(key string) error {
	return nil
}

func (c *DummyDriver) Close() error {
	return nil
}

func (c *DummyDriver) Connection() interface{} {
	return nil
}

func OpenDummyDriver(value string, o Options) Driver {
	return &DummyDriver{}
}

func init() {
	Register("dummy", OpenDummyDriver)
}
