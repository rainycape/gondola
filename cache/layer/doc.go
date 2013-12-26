// Package layer implements a cache layer which allows
// caching of complete responses.
//
// Use New to initialize a new Layer and then call Wrap
// on any app.Handler to obtain a new app.Handler wrapped
// by the Layer.
//
// A Mediator indicates the layer if a response should be
// cached and for how long, as well as indicating which requests
// should bypass the Layer.
//
// This package provides the SimpleMediator, which implements the
// Mediator protocol with enough knobs to satisty most common needs.
// Users with more advanced requirements should write their own Mediator
// implementation.
//
//  cache, err := myapp.Cache()
//  if err != nil {
//	panic(err)
//  }
//  layer := layer.New(cache.Cache, &layer.SimpleMediator{Expiration:600})
//  myapp.Handle("/something/", layer.Wrap(MyHandler))
package layer
