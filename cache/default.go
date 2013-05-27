package cache

var (
	defaultCache Config
)

func SetDefault(c string) error {
	config, err := ParseConfig(c)
	if err != nil {
		return err
	}
	defaultCache = *config
	return nil
}

func Default() string {
	return defaultCache.String()
}
