package bootstrap3

type Size int

const (
	ExtraSmall = iota - 2
	Small
	Medium
	Large
)

func (s Size) String() string {
	switch s {
	case ExtraSmall:
		return "xs"
	case Small:
		return "sm"
	case Medium:
		return "md"
	case Large:
		return "lg"
	}
	return ""
}

type Alignment int

const (
	Left Alignment = iota
	Center
	Right
)
