// Package ads implements a small reusable app for easily
// displaying adversitesements from different providers.
//
// To use this application, import it somewhere in your code,
// include it into your main app and set the PublisherID field
// in the providers you plan to use.
//
//  import (
//	...
//	"gnd.la/apps/ads"
//	...
//  )
//
//  ...
//  ads.AdSense.PublisherID = "ca-pub-123456"
//  App.Include("/ads", ads.App, "")
//
// Then, from your templates you can use the template function "ad". See
// Ad for its documentation.
//
// Additionaly, this app also exports a Javascript API, but using it is not required.
// The only exposed function is.
//
//  Ads.Init(options)
//
// If you use this function, you should usually call it from a handler attached to
// the document.ready event:
//
//  $(document).ready(function() {
//	Ads.Init({
//	    Google: true,
//	    Chitika: false
//	});
//  });
//
// The options parameter allows selectively disabling ads from one or several
// providers (e.g. displaying ads for free users and hiding them for paid
// users, or showing ads from each provider 50% of time). Finally, if this
// function has not been called by the time window.load trigers, it will be
// automatically called with no options, loading all the ads present in the
// page. Note that ads hidden via CSS (e.g. using media queries) will
// always be removed.
package ads
