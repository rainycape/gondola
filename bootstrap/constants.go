package bootstrap

type Size int

const (
	ExtraSmall = iota - 2
	Small
	Medium
	Large
)

type Alignment int

const (
	Left Alignment = iota
	Center
	Right
)
