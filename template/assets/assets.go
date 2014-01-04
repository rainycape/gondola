package assets

import (
	"fmt"
)

const (
	analyticsScript = `<script>(function(i,s,o,g,r,a,m){i['GoogleAnalyticsObject']=r;i[r]=i[r]||function(){
	(i[r].q=i[r].q||[]).push(arguments)},i[r].l=1*new Date();a=s.createElement(o),
	m=s.getElementsByTagName(o)[0];a.async=1;a.src=g;m.parentNode.insertBefore(a,m)
	})(window,document,'script','//www.google-analytics.com/analytics.js','ga');
	ga('create', %s);
	ga('send', 'pageview');</script>`
)

func googleAnalytics(m *Manager, names []string, options Options) ([]*Asset, error) {
	if len(names) != 1 && len(names) != 2 {
		return nil, fmt.Errorf("analytics requires either 1 or 2 arguments (either \"UA-XXXXXX-YY, mysite.com\" or just \"UA-XXXXXX-YY\" - without quotes in both cases")
	}
	key := names[0]
	if key == "" {
		return nil, nil
	}
	var arg string
	if len(names) == 2 {
		arg = fmt.Sprintf("'%s', '%s'", key, names[1])
	} else {
		arg = fmt.Sprintf("'%s'", key)
	}
	return []*Asset{
		&Asset{
			Name:     "google-analytics.js",
			Position: Bottom,
			HTML:     fmt.Sprintf(analyticsScript, arg),
		},
	}, nil
}

func init() {
	Register("analytics", googleAnalytics)
}
