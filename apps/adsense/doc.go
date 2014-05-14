// Package adsense implements a small reusable app for
// displaying responsive AdSense ads.
//
// To use this application, import it somewhere in your code,
// include it into your main app and set Publisher to your
// AdSense publisher ID.
//
//  import (
//	...
//	"gnd.la/apps/adsense"
//	...
//  )
//
//  ...
//  adsense.Publisher = "ca-pub-123456"
//  App.Include("/adsense", adsense.App, "")
//
// Then, from your templates you can invoke any of the templates exported
// by this app. All of them receive a single argument, the ad slot ID.
//
//  {{ template "AdSense|Responsive" "123456789" }}
//
// You might include any of the templates any number of times (keep
// in mind that AdSense ToS limit the number of ads per page). Note that
// all the templates exported by this package require jQuery to be loaded
// in the same page.
//
// The available templates are:
//
//  - AdSense|Responsive: Displays a responsive ad at the point where the template
//	is included.
//
//  - Adsense|ResponsiveFixed: Displays a responsive ad fixed at the bottom. The ad
//	also includes a small hide/show button, to avoid covering any important parts
//	of the page that might be innacessible otherwise.
//
// Additionaly, this app also exports a Javascript API, but using it is not required.
// The only exposed function is.
//
//  AdSense.Init(disabled)
//
// If you use this function, you should usually call it from a handler attached to
// the document.ready event:
//
//  $(document).ready(function() {
//	AdSense.Init();
//  });
//
// The disabled parameter allows selectively disabling the ads in some causes (e.g.
// displaying them for free users and hiding them for paid users). Finally, if this
// function has not been called by the time window.load trigers, it will be automatically
// called with disabled set to false.
package adsense
