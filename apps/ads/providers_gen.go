package ads

// AUTOMATICALLY GENERATED WITH /tmp/go-build450725549/command-line-arguments/_obj/exe/gen -- DO NOT EDIT!

var (
	// AdSense implements a Provider which displays Google's AdSense ads.
	// For more information, see http://www.google.com/adsense/.
	//
	// AdSense supports the following ad sizes:
	// Size120x90, Size120x240, Size120x600,
	// Size125x125, Size160x90, Size160x600, Size180x90,
	// Size180x150, Size200x90, Size200x200, Size234x60,
	// Size250x250, Size300x250, Size300x600,
	// Size300x1050, Size320x50, Size320x100,
	// Size336x280, Size468x15, Size468x60, Size728x15,
	// Size728x90, Size970x90, Size970x250,
	// SizeResponsive.
	AdSense = &Provider{
		Name:         "AdSense",
		URL:          "http://www.google.com/adsense/",
		script:       "//pagead2.googlesyndication.com/pagead/js/adsbygoogle.js",
		defaultSlot:  "",
		requiresSlot: true,
		responsive:   true,
		render:       renderAdSenseAd,
		className:    "ads-adsense",
	}
	// Chitika implements a Provider which displays Chitika ads.
	// For more information, see http://www.chitika.com.
	//
	// Chitika supports the following ad sizes:
	// Size120x600, Size160x160, Size160x600,
	// Size200x200, Size250x250, Size300x150,
	// Size300x250, Size300x600, Size336x280,
	// Size468x60, Size468x180, Size468x250,
	// Size500x200, Size500x250, Size550x120,
	// Size550x250, Size728x90, SizeResponsive.
	Chitika = &Provider{
		Name:         "Chitika",
		URL:          "http://www.chitika.com",
		script:       "//cdn.chitika.net/getads.js",
		defaultSlot:  "Chitika Default",
		requiresSlot: false,
		responsive:   true,
		render:       renderChitikaAd,
		className:    "ads-chitika",
	}

	providers = []*Provider{
		AdSense,
		Chitika,
	}
)

func (p *Provider) supportsSize(s Size) bool {
	switch p {
	case AdSense:
		return s == Size120x90 || s == Size120x240 || s == Size120x600 || s == Size125x125 || s == Size160x90 || s == Size160x600 || s == Size180x90 || s == Size180x150 || s == Size200x90 || s == Size200x200 || s == Size234x60 || s == Size250x250 || s == Size300x250 || s == Size300x600 || s == Size300x1050 || s == Size320x50 || s == Size320x100 || s == Size336x280 || s == Size468x15 || s == Size468x60 || s == Size728x15 || s == Size728x90 || s == Size970x90 || s == Size970x250 || s == SizeResponsive
	case Chitika:
		return s == Size120x600 || s == Size160x160 || s == Size160x600 || s == Size200x200 || s == Size250x250 || s == Size300x150 || s == Size300x250 || s == Size300x600 || s == Size336x280 || s == Size468x60 || s == Size468x180 || s == Size468x250 || s == Size500x200 || s == Size500x250 || s == Size550x120 || s == Size550x250 || s == Size728x90 || s == SizeResponsive

	}
	return false
}
