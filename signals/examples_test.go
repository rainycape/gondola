package signals_test

import (
	"fmt"
	"time"

	"gnd.la/signals"
)

// Message is exported, so callers can use the type emitted
// by the signal.
type Message struct {
	Text      string
	Timestamp time.Time
}

// messageSignal is unexported, so only this package can create
// instances. However, it contains exported methods to allow
// listening and or emitting signals from calling packages.
type messageSignal struct {
	s *signals.Signal
}

// Listen is exported, so callers can listen to this signal.
func (s *messageSignal) Listen(handler func(*Message)) signals.Listener {
	return s.s.Listen(func(data interface{}) {
		handler(data.(*Message))
	})
}

// emit is unexported, so only this package can emit the signal. Note that
// a package might choose to export Emit too.
func (s *messageSignal) emit(msg *Message) {
	s.s.Emit(msg)
}

// Signals is exported, so package callers can access it and its fields
// as e.g. pkg.Signals.MessageReceived.Listen(...)
var Signals = struct {
	MessageReceived *messageSignal
}{
	&messageSignal{signals.New("message-received")},
}

func ExampleSignal() {
	Signals.MessageReceived.Listen(func(msg *Message) {
		fmt.Println(msg.Text, msg.Timestamp.Unix())
	})
	Signals.MessageReceived.emit(&Message{
		Text:      "hello",
		Timestamp: time.Unix(42, 0),
	})
	// Output: hello 42
}

func ExampleRemove() {
	const (
		signame = "mysignal"
		payload = "emitted"
	)
	listener := signals.Listen(signame, func(_ string, data interface{}) {
		fmt.Println(data)
	})
	signals.Emit(signame, payload)
	listener.Remove()
	// Emitting again won't cause the listener to be called
	// since the listener was removed.
	signals.Emit(signame, payload)
	// Output: emitted
}
