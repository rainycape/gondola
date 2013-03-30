package log

type Writer interface {
	Write(LLevel, int, []byte) (int, error)
	Level() LLevel
}
