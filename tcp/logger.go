package tcp

type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debug(format string)
	Info(format string)
	Warn(format string)
	Error(format string)
}
