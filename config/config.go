package config

type Config struct {
	Debug            bool   `help:"Enable debug mode"`
	Port             int    `default:"8888" help:"Port to listen on"`
	CacheUrl         string `help:"Default cache URL"`
	CookieHashKey    string `help:"Key used for hashing cookies"`
	CookieEncryptKey string `help:"Key used for encrypting cookies"`
}
