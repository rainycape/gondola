package ads

import (
	"bytes"
	"fmt"

	"gnd.la/app"
	"gnd.la/template"
)

// Ad returns the HTML for an ad with the given provider, slot ID, size and options.
// This function is exported as a template function named "ad". Note that the first
// argument (the *app.Context) is implicit and passed by the template VM.
//
// Some examples
//
// To insert a 728x90 ad, use:
//
//  {{ ad @Ads.AdSense "123456789" @Ads.Sizes.S728x90 }}
//
// To insert a responsive ad fixed at the bottom, use:
//
//  {{ ad @Ads.AdSense "123456789" @Ads.Sizes.Responsive @Ads.Fixed @Ads.Bottom }}
//
func Ad(ctx *app.Context, provider *Provider, slot string, s Size, options ...string) (template.HTML, error) {
	if provider.PublisherID == "" {
		return template.HTML(""), fmt.Errorf("provider %s has no publisher ID", provider.Name)
	}
	if slot == "" {
		if provider.requiresSlot {
			return template.HTML(""), fmt.Errorf("provider %s requires a slot ID", provider.Name)
		}
		slot = provider.defaultSlot
	}
	if !provider.supportsSize(s) {
		return template.HTML(""), fmt.Errorf("provider %s does not support ad size %s", provider.Name, s)
	}
	var buf bytes.Buffer
	// Container
	buf.WriteString("<div class=\"ads-container")
	if s == SizeResponsive {
		buf.WriteString(" ads-responsive")
	}
	if hasOption(options, Fixed) {
		buf.WriteString(" ads-fixed")
		if hasOption(options, Top) {
			buf.WriteString(" ads-fixed-top")
		} else if hasOption(options, Bottom) {
			buf.WriteString(" ads-fixed-bottom")
		} else {
			return template.HTML(""), fmt.Errorf("Fixed requires using either Top or Bottom (options: %v)", options)
		}
	}
	buf.WriteString("\">")
	// Box
	buf.WriteString("<div class=\"")
	buf.WriteString(provider.className)
	buf.WriteString(" ads-box ads-top-box\"")
	if s != SizeResponsive {
		fmt.Fprintf(&buf, " style=\"display: %s; width: %dpx; height: %dpx;\"", s.Display(), s.Width(), s.Height())
	}
	buf.WriteByte('>')
	// Hide/Show button, if fixed
	if hasOption(options, Fixed) {
		//  Use display:none inlined, so it's hidden when using adblock
		buf.WriteString("<a class=\"ads-hide-button\" style=\"display:none;\" href=\"#\"><span class=\"ads-hide\">")
		buf.WriteString(ctx.Tc("Ads", "hide"))
		buf.WriteString("</span><span class=\"ads-show\">")
		buf.WriteString(ctx.Tc("Ads", "show"))
		buf.WriteString("</span></a>")
	}
	// Render the ad
	provider.render(&buf, ctx, provider, slot, s, options)
	// Close box and container
	buf.WriteString("</div></div>")
	return template.HTML(buf.String()), nil
}

func hasOption(options []string, opt string) bool {
	for _, v := range options {
		if v == opt {
			return true
		}
	}
	return false
}

func init() {
	template.AddFuncs(template.FuncMap{
		"!ad": Ad,
	})
}
