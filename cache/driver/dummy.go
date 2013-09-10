package driver

var ddrv *DummyDriver

// DummyDriver implements a dummy cache, which doesn't store
// any information and never returns an object.
type DummyDriver struct {
}

func (d *DummyDriver) Set(key string, b []byte, timeout int) error {
	return nil
}

func (d *DummyDriver) Get(key string) ([]byte, error) {
	return nil, nil
}

func (d *DummyDriver) GetMulti(keys []string) (map[string][]byte, error) {
	return nil, nil
}

func (d *DummyDriver) Delete(key string) error {
	return nil
}

func (d *DummyDriver) Close() error {
	return nil
}

func (d *DummyDriver) Connection() interface{} {
	return nil
}

func openDummyDriver(value string, o Options) (Driver, error) {
	if ddrv == nil {
		// No locking, since the worst thing that might
		// happen is we end up with several DummyDriver
		// instances and some of them will be collected
		// by the GC when they're no longer used. Avoid
		// allocating this in init(), since there's no
		// no reason to allocate a few bytes if the
		// app is not using the DummyDriver (which will
		// happen most of the time).
		ddrv = &DummyDriver{}
	}
	return ddrv, nil
}

func init() {
	Register("dummy", openDummyDriver)
}
