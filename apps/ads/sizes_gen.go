package ads

// AUTOMATICALLY GENERATED WITH /tmp/go-build450725549/command-line-arguments/_obj/exe/gen -- DO NOT EDIT!

import "fmt"

const (
	Size120x240  Size = Size(120<<16 | 240)
	Size120x600  Size = Size(120<<16 | 600)
	Size120x90   Size = Size(120<<16 | 90)
	Size125x125  Size = Size(125<<16 | 125)
	Size160x160  Size = Size(160<<16 | 160)
	Size160x600  Size = Size(160<<16 | 600)
	Size160x90   Size = Size(160<<16 | 90)
	Size180x150  Size = Size(180<<16 | 150)
	Size180x90   Size = Size(180<<16 | 90)
	Size200x200  Size = Size(200<<16 | 200)
	Size200x90   Size = Size(200<<16 | 90)
	Size234x60   Size = Size(234<<16 | 60)
	Size250x250  Size = Size(250<<16 | 250)
	Size300x1050 Size = Size(300<<16 | 1050)
	Size300x150  Size = Size(300<<16 | 150)
	Size300x250  Size = Size(300<<16 | 250)
	Size300x600  Size = Size(300<<16 | 600)
	Size320x100  Size = Size(320<<16 | 100)
	Size320x50   Size = Size(320<<16 | 50)
	Size336x280  Size = Size(336<<16 | 280)
	Size468x15   Size = Size(468<<16 | 15)
	Size468x180  Size = Size(468<<16 | 180)
	Size468x250  Size = Size(468<<16 | 250)
	Size468x60   Size = Size(468<<16 | 60)
	Size500x200  Size = Size(500<<16 | 200)
	Size500x250  Size = Size(500<<16 | 250)
	Size550x120  Size = Size(550<<16 | 120)
	Size550x250  Size = Size(550<<16 | 250)
	Size728x15   Size = Size(728<<16 | 15)
	Size728x90   Size = Size(728<<16 | 90)
	Size970x250  Size = Size(970<<16 | 250)
	Size970x90   Size = Size(970<<16 | 90)
)

var sizes = map[string]Size{
	"S120x240":   Size120x240,
	"S120x600":   Size120x600,
	"S120x90":    Size120x90,
	"S125x125":   Size125x125,
	"S160x160":   Size160x160,
	"S160x600":   Size160x600,
	"S160x90":    Size160x90,
	"S180x150":   Size180x150,
	"S180x90":    Size180x90,
	"S200x200":   Size200x200,
	"S200x90":    Size200x90,
	"S234x60":    Size234x60,
	"S250x250":   Size250x250,
	"S300x1050":  Size300x1050,
	"S300x150":   Size300x150,
	"S300x250":   Size300x250,
	"S300x600":   Size300x600,
	"S320x100":   Size320x100,
	"S320x50":    Size320x50,
	"S336x280":   Size336x280,
	"S468x15":    Size468x15,
	"S468x180":   Size468x180,
	"S468x250":   Size468x250,
	"S468x60":    Size468x60,
	"S500x200":   Size500x200,
	"S500x250":   Size500x250,
	"S550x120":   Size550x120,
	"S550x250":   Size550x250,
	"S728x15":    Size728x15,
	"S728x90":    Size728x90,
	"S970x250":   Size970x250,
	"S970x90":    Size970x90,
	"Responsive": SizeResponsive,
}

func (s Size) String() string {
	switch s {
	case Size120x240:
		return "120x240"
	case Size120x600:
		return "120x600"
	case Size120x90:
		return "120x90"
	case Size125x125:
		return "125x125"
	case Size160x160:
		return "160x160"
	case Size160x600:
		return "160x600"
	case Size160x90:
		return "160x90"
	case Size180x150:
		return "180x150"
	case Size180x90:
		return "180x90"
	case Size200x200:
		return "200x200"
	case Size200x90:
		return "200x90"
	case Size234x60:
		return "234x60"
	case Size250x250:
		return "250x250"
	case Size300x1050:
		return "300x1050"
	case Size300x150:
		return "300x150"
	case Size300x250:
		return "300x250"
	case Size300x600:
		return "300x600"
	case Size320x100:
		return "320x100"
	case Size320x50:
		return "320x50"
	case Size336x280:
		return "336x280"
	case Size468x15:
		return "468x15"
	case Size468x180:
		return "468x180"
	case Size468x250:
		return "468x250"
	case Size468x60:
		return "468x60"
	case Size500x200:
		return "500x200"
	case Size500x250:
		return "500x250"
	case Size550x120:
		return "550x120"
	case Size550x250:
		return "550x250"
	case Size728x15:
		return "728x15"
	case Size728x90:
		return "728x90"
	case Size970x250:
		return "970x250"
	case Size970x90:
		return "970x90"
	case SizeResponsive:
		return "responsive"

	}
	return fmt.Sprintf("invalid size %d", int(s))
}
