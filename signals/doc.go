// Package signals implements functions for emitting and receiving
// synchronous signals.
//
// This package has a high level API, which is the recommended usage,
// based on the Signal type. Users should define typed signals for
// each required argument type, using non-exported types but with
// exported methods, the use a public anonymous structure to export
// all signals. This allows packages to export either listen/emit or
// listen only signals.
//
// First, define your typed signal:
//
//  // Message is exported, so callers can use the data type emitted
//  // by the signal.
//  type Message struct {
//  ...
//  }
//
//  // messageSignal is unexported, but contains exported methods
//  type messageSignal struct {
//      s *signals.Signal
//  }
//
//  func(s *messageSignal) Listen(handler func(*Message)) signals.Listener {
//      return s.s.Listen(func(data interface{}) { handler(data.(*Message))})
//  })
package signals
