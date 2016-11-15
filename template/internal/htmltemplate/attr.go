// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"strings"
)

// attrTypeMap[n] describes the value of the given attribute.
// If an attribute affects (or can mask) the encoding or interpretation of
// other content, or affects the contents, idempotency, or credentials of a
// network message, then the value in this map is contentTypeUnsafe.
// This map is derived from HTML5, specifically
// http://www.w3.org/TR/html5/Overview.html#attributes-1
// as well as "%URI"-typed attributes from
// http://www.w3.org/TR/html4/index/attributes.html
var attrTypeMap = map[string]ContentType{
	"accept":          ContentTypePlain,
	"accept-charset":  contentTypeUnsafe,
	"action":          ContentTypeURL,
	"alt":             ContentTypePlain,
	"archive":         ContentTypeURL,
	"async":           contentTypeUnsafe,
	"autocomplete":    ContentTypePlain,
	"autofocus":       ContentTypePlain,
	"autoplay":        ContentTypePlain,
	"background":      ContentTypeURL,
	"border":          ContentTypePlain,
	"checked":         ContentTypePlain,
	"cite":            ContentTypeURL,
	"challenge":       contentTypeUnsafe,
	"charset":         contentTypeUnsafe,
	"class":           ContentTypePlain,
	"classid":         ContentTypeURL,
	"codebase":        ContentTypeURL,
	"cols":            ContentTypePlain,
	"colspan":         ContentTypePlain,
	"content":         contentTypeUnsafe,
	"contenteditable": ContentTypePlain,
	"contextmenu":     ContentTypePlain,
	"controls":        ContentTypePlain,
	"coords":          ContentTypePlain,
	"crossorigin":     contentTypeUnsafe,
	"data":            ContentTypeURL,
	"datetime":        ContentTypePlain,
	"default":         ContentTypePlain,
	"defer":           contentTypeUnsafe,
	"dir":             ContentTypePlain,
	"dirname":         ContentTypePlain,
	"disabled":        ContentTypePlain,
	"draggable":       ContentTypePlain,
	"dropzone":        ContentTypePlain,
	"enctype":         contentTypeUnsafe,
	"for":             ContentTypePlain,
	"form":            contentTypeUnsafe,
	"formaction":      ContentTypeURL,
	"formenctype":     contentTypeUnsafe,
	"formmethod":      contentTypeUnsafe,
	"formnovalidate":  contentTypeUnsafe,
	"formtarget":      ContentTypePlain,
	"headers":         ContentTypePlain,
	"height":          ContentTypePlain,
	"hidden":          ContentTypePlain,
	"high":            ContentTypePlain,
	"href":            ContentTypeURL,
	"hreflang":        ContentTypePlain,
	"http-equiv":      contentTypeUnsafe,
	"icon":            ContentTypeURL,
	"id":              ContentTypePlain,
	"ismap":           ContentTypePlain,
	"keytype":         contentTypeUnsafe,
	"kind":            ContentTypePlain,
	"label":           ContentTypePlain,
	"lang":            ContentTypePlain,
	"language":        contentTypeUnsafe,
	"list":            ContentTypePlain,
	"longdesc":        ContentTypeURL,
	"loop":            ContentTypePlain,
	"low":             ContentTypePlain,
	"manifest":        ContentTypeURL,
	"max":             ContentTypePlain,
	"maxlength":       ContentTypePlain,
	"media":           ContentTypePlain,
	"mediagroup":      ContentTypePlain,
	"method":          contentTypeUnsafe,
	"min":             ContentTypePlain,
	"multiple":        ContentTypePlain,
	"name":            ContentTypePlain,
	"novalidate":      contentTypeUnsafe,
	// Skip handler names from
	// http://www.w3.org/TR/html5/webappapis.html#event-handlers-on-elements,-document-objects,-and-window-objects
	// since we have special handling in attrType.
	"open":        ContentTypePlain,
	"optimum":     ContentTypePlain,
	"pattern":     contentTypeUnsafe,
	"placeholder": ContentTypePlain,
	"poster":      ContentTypeURL,
	"profile":     ContentTypeURL,
	"preload":     ContentTypePlain,
	"pubdate":     ContentTypePlain,
	"radiogroup":  ContentTypePlain,
	"readonly":    ContentTypePlain,
	"rel":         contentTypeUnsafe,
	"required":    ContentTypePlain,
	"reversed":    ContentTypePlain,
	"rows":        ContentTypePlain,
	"rowspan":     ContentTypePlain,
	"sandbox":     contentTypeUnsafe,
	"spellcheck":  ContentTypePlain,
	"scope":       ContentTypePlain,
	"scoped":      ContentTypePlain,
	"seamless":    ContentTypePlain,
	"selected":    ContentTypePlain,
	"shape":       ContentTypePlain,
	"size":        ContentTypePlain,
	"sizes":       ContentTypePlain,
	"span":        ContentTypePlain,
	"src":         ContentTypeURL,
	"srcdoc":      ContentTypeHTML,
	"srclang":     ContentTypePlain,
	"start":       ContentTypePlain,
	"step":        ContentTypePlain,
	"style":       ContentTypeCSS,
	"tabindex":    ContentTypePlain,
	"target":      ContentTypePlain,
	"title":       ContentTypePlain,
	"type":        contentTypeUnsafe,
	"usemap":      ContentTypeURL,
	"value":       contentTypeUnsafe,
	"width":       ContentTypePlain,
	"wrap":        ContentTypePlain,
	"xmlns":       ContentTypeURL,
}

// attrType returns a conservative (upper-bound on authority) guess at the
// type of the named attribute.
func attrType(name string) ContentType {
	name = strings.ToLower(name)
	if strings.HasPrefix(name, "data-") {
		// Strip data- so that custom attribute heuristics below are
		// widely applied.
		// Treat data-action as URL below.
		name = name[5:]
	} else if colon := strings.IndexRune(name, ':'); colon != -1 {
		if name[:colon] == "xmlns" {
			return ContentTypeURL
		}
		// Treat svg:href and xlink:href as href below.
		name = name[colon+1:]
	}
	if t, ok := attrTypeMap[name]; ok {
		return t
	}
	// Treat partial event handler names as script.
	if strings.HasPrefix(name, "on") {
		return ContentTypeJS
	}

	// Heuristics to prevent "javascript:..." injection in custom
	// data attributes and custom attributes like g:tweetUrl.
	// http://www.w3.org/TR/html5/dom.html#embedding-custom-non-visible-data-with-the-data-*-attributes
	// "Custom data attributes are intended to store custom data
	//  private to the page or application, for which there are no
	//  more appropriate attributes or elements."
	// Developers seem to store URL content in data URLs that start
	// or end with "URI" or "URL".
	if strings.Contains(name, "src") ||
		strings.Contains(name, "uri") ||
		strings.Contains(name, "url") {
		return ContentTypeURL
	}
	return ContentTypePlain
}
