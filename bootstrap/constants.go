package bootstrap

type Size int

const (
	SizeMini Size = iota - 2
	SizeSmall
	SizeMedium
	SizeLarge
)

type Alignment int

const (
	AlignmentLeft Alignment = iota
	AlignmentCenter
	AlignmentRight
)
