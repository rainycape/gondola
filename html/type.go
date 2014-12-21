package html

type Type int

const (
	TypeTag Type = iota
	TypeText
	TypeAny Type = -1
)
