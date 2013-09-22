package orm

import (
	"gnd.la/log"
)

type Logger interface {
	SetLogger(*log.Logger)
}
