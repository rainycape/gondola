package log

type LLevel int

const (
	LDebug LLevel = iota
	LInfo
	LWarning
	LError
	LPanic
	LFatal
	LNone
	LDefault = LInfo
)

func (l LLevel) String() string {
	switch l {
	case LDebug:
		return "Debug"
	case LInfo:
		return "Info"
	case LWarning:
		return "Warning"
	case LError:
		return "Error"
	case LPanic:
		return "Panic"
	case LFatal:
		return "Fatal"
	case LNone:
		return "None"
	}
	return "Unknown"
}

func (l LLevel) Initial() string {
	switch l {
	case LDebug:
		return "D"
	case LInfo:
		return "I"
	case LWarning:
		return "W"
	case LError:
		return "E"
	case LPanic:
		return "P"
	case LFatal:
		return "F"
	case LNone:
		return "N"
	}
	return "U"
}

func (l LLevel) Colorcode() string {
	switch l {
	case LDebug:
		return "0;32" // Green
	case LInfo:
		return "1;34" // Light Blue
	case LWarning:
		return "1;33" // Yellow
	case LError:
		return "1;31" // Light Red
	case LPanic, LFatal:
		return "0;31" // Red
	}
	return "1;37" // White
}

var (
	colorDebug   []byte
	colorInfo    []byte
	colorWarning []byte
	colorError   []byte
	colorPanic   []byte
	colorWhite   []byte
)

func (l LLevel) colorBeginBytes() []byte {
	switch l {
	case LDebug:
		return colorDebug
	case LInfo:
		return colorInfo
	case LWarning:
		return colorWarning
	case LError:
		return colorError
	case LPanic, LFatal:
		return colorPanic
	}
	return colorWhite
}

func init() {
	makeColor := func(l LLevel) []byte {
		return []byte("\x1b\x5b" + l.Colorcode() + "m")
	}
	colorDebug = makeColor(LDebug)
	colorInfo = makeColor(LInfo)
	colorWarning = makeColor(LWarning)
	colorError = makeColor(LError)
	colorPanic = makeColor(LPanic)
	colorWhite = makeColor(LNone)
}
