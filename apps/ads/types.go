package ads

const (
	// Top and Bottom represent the ad position when it's displayed
	// as Fixed ad.
	Top    = "Top"
	Bottom = "Bottom"

	// Fixed displays the ad in a fixed position, either
	// at the Top or the Bottom. When using Fixed as an
	// option, either Top or Bottom must be used too.
	Fixed = "Fixed"
)

// Size represents the ad size. Use the available constants
// in this package to
type Size uint32

// Width returns the ad width.
func (s Size) Width() int {
	return int(s >> 16)
}

// Height returns the ad height.
func (s Size) Height() int {
	return int(s & 0xFFFF)
}

// Display returns the default display mode for the
// ad size, either inline-block or block.
func (s Size) Display() string {
	if s.Height() <= 30 && s != SizeResponsive {
		return "inline-block"
	}
	return "block"
}

const (
	// SizeResponsive generates a responsive ad, which
	// adapts its size to the web browser screen size.
	SizeResponsive Size = 1
)

// Provider represents an ad provider. Users should not
// create their own providers, but use the available ones.
// See AdSense and Chitika.
type Provider struct {
	Name         string
	URL          string
	PublisherID  string
	script       string
	className    string
	defaultSlot  string
	requiresSlot bool
	responsive   bool
	render       renderFunc
}
