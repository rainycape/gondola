package bootstrap

type Size int

const (
	SizeExtraSmall = iota - 1
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
