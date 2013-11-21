package assets

import (
	"fmt"
)

const (
	analyticsScript = ` <script>(function(i,s,o,g,r,a,m){i['GoogleAnalyticsObject']=r;i[r]=i[r]||function(){
	(i[r].q=i[r].q||[]).push(arguments)},i[r].l=1*new Date();a=s.createElement(o),
	m=s.getElementsByTagName(o)[0];a.async=1;a.src=g;m.parentNode.insertBefore(a,m)
	})(window,document,'script','//www.google-analytics.com/analytics.js','ga');
	ga('create', '%s');
	ga('send', 'pageview');</script>`
)

func googleAnalytics(m Manager, names []string, options Options) ([]Asset, error) {
	key := names[0]
	if key == "" {
		return nil, nil
	}
	return []Asset{
		&Script{
			Common: Common{
				Manager: m,
				Name:    "google-analytics.js",
			},
			Position: Bottom,
			Script:   fmt.Sprintf(analyticsScript, key),
		},
	}, nil
}

func init() {
	Register("analytics", singleParser(googleAnalytics))
}
