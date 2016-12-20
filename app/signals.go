package app

import (
	"gnd.la/signals"
)

type appSignal struct {
	signal *signals.Signal
}

func (s *appSignal) Listen(handler func(app *App)) signals.Listener {
	return s.signal.Listen(func(data interface{}) {
		handler(data.(*App))
	})
}

func (s *appSignal) emit(app *App) {
	s.signal.Emit(app)
}

// Signals declares the signals emitted by this package. See
// gnd.la/signals for more information.
var Signals = struct {
	// WillListen is emitted just before a *gnd.la/app.App will
	// start listening.
	WillListen *appSignal
	// DidListen is emitted after a *gnd.la/app.App starts
	// listening.
	DidListen *appSignal
	// WillPrepare is emitted at the beginning of App.Prepare.
	WillPrepare *appSignal
	// DidPrepare is emitted when App.Prepare ends without errors.
	DidPrepare *appSignal
}{
	WillListen:  &appSignal{signals.New("gnd.la/app.will-listen")},
	DidListen:   &appSignal{signals.New("gnd.la/app.did-listen")},
	WillPrepare: &appSignal{signals.New("gnd.la/app.will-prepare")},
	DidPrepare:  &appSignal{signals.New("gnd.la/app.did-prepare")},
}
