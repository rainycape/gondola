// Package signals implements functions for emitting and receiving
// synchronous signals.
//
// A signal is always sent by a single sender, but can reach an
// arbitrary number of receivers.
//
// Users should define typed signals for each required argument type
// using non-exported types but with exported methods. Then, use a
// public anonymous structure to export all signals. This allows
// packages to export either listen/emit or listen only signals.
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
//  // Listen is exported, so other packages can listen for this signal
//  func(s *messageSignal) Listen(handler func(*Message)) signals.Listener {
//      return s.s.Listen(func(data interface{}) { handler(data.(*Message))})
//  })
//
//  // emit is unexported, so only this package can emit the signal. If
//  // other packages should be able to emit the signal, just export it
//  // as Emit().
//  func(s *messageSignal) emit(msg *Message) {
//      s.s.Emit(msg)
//  }
//
// Then, define a package level exported struct which will hold all your
// package signals:
//
//  var Signals = struct{
//      MessageReceived *messageSignal
//      MessageSent *messageSignal
//  }{
//      &messageSignal{signals.New("message-received")},
//      &messageSignal{signals.New("message-sent")},
//  }
//
// When you want to emit a signal, just call its emit method:
//
//  Signals.MessageReceived.emit(&Message{...})
//
// On the other hand, listeners will be able to register themselves like:
//
//  your.package.Signals.MessageReceived.Listen(func(msg *your.package.Message{
//    ...
//  }))
package signals
