package ads

var (
	// Wheter to show a shadow around fixed ads.
	Shadow  bool
	AdSense struct {
		// Publisher must be set to your AdSense publisher ID
		// e.g. ca-pub-123456.
		Publisher string
	}
)
