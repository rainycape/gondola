package orm

import (
	"gondola/log"
)

type Logger interface {
	SetLogger(*log.Logger)
}
