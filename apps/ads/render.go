package ads

import (
	"bytes"
	"fmt"

	"gnd.la/app"
)

type renderFunc func(buf *bytes.Buffer, ctx *app.Context, provider *Provider, slot string, s Size, options []string)

func renderAdSenseAd(buf *bytes.Buffer, ctx *app.Context, provider *Provider, slot string, s Size, options []string) {
	buf.WriteString("<ins class=\"adsbygoogle ads-box\" style=\"")
	if s == SizeResponsive {
		buf.WriteString("display:block\"")
	} else {
		fmt.Fprintf(buf, "display: %s; width: %dpx; height: %dpx;\"", s.Display(), s.Width(), s.Height())
	}
	buf.WriteString(" data-ad-client=\"")
	buf.WriteString(provider.PublisherID)
	buf.WriteString("\" data-ad-slot=\"")
	buf.WriteString(slot)
	//buf.WriteString("\" data-ad-format=\"auto\"></ins><script>(adsbygoogle = window.adsbygoogle || []).push({});</script>")
	buf.WriteString("\"></ins><script>(adsbygoogle = window.adsbygoogle || []).push({});</script>")
}

func renderChitikaAd(buf *bytes.Buffer, ctx *app.Context, provider *Provider, slot string, s Size, options []string) {
	buf.WriteString("<div data-publisher=\"")
	buf.WriteString(provider.PublisherID)
	buf.WriteString("\" data-sid=\"")
	buf.WriteString(slot)
	buf.WriteString("\"></div>")
}
