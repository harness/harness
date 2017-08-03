// +build !windows

package zerolog

import "io"

// SyslogWriter is an interface matching a syslog.Writer struct.
type SyslogWriter interface {
	io.Writer
	Debug(m string) error
	Info(m string) error
	Warning(m string) error
	Err(m string) error
	Emerg(m string) error
	Crit(m string) error
}

type syslogWriter struct {
	w SyslogWriter
}

// SyslogLevelWriter wraps a SyslogWriter and call the right syslog level
// method matching the zerolog level.
func SyslogLevelWriter(w SyslogWriter) LevelWriter {
	return syslogWriter{w}
}

func (sw syslogWriter) Write(p []byte) (n int, err error) {
	return sw.w.Write(p)
}

// WriteLevel implements LevelWriter interface.
func (sw syslogWriter) WriteLevel(level Level, p []byte) (n int, err error) {
	switch level {
	case DebugLevel:
		err = sw.w.Debug(string(p))
	case InfoLevel:
		err = sw.w.Info(string(p))
	case WarnLevel:
		err = sw.w.Warning(string(p))
	case ErrorLevel:
		err = sw.w.Err(string(p))
	case FatalLevel:
		err = sw.w.Emerg(string(p))
	case PanicLevel:
		err = sw.w.Crit(string(p))
	default:
		panic("invalid level")
	}
	n = len(p)
	return
}
