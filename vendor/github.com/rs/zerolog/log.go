// Package zerolog provides a lightweight logging library dedicated to JSON logging.
//
// A global Logger can be use for simple logging:
//
//     import "github.com/rs/zerolog/log"
//
//     log.Info().Msg("hello world")
//     // Output: {"time":1494567715,"level":"info","message":"hello world"}
//
// NOTE: To import the global logger, import the "log" subpackage "github.com/rs/zerolog/log".
//
// Fields can be added to log messages:
//
//     log.Info().Str("foo", "bar").Msg("hello world")
//     // Output: {"time":1494567715,"level":"info","message":"hello world","foo":"bar"}
//
// Create logger instance to manage different outputs:
//
//     logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
//     logger.Info().
//            Str("foo", "bar").
//            Msg("hello world")
//     // Output: {"time":1494567715,"level":"info","message":"hello world","foo":"bar"}
//
// Sub-loggers let you chain loggers with additional context:
//
//     sublogger := log.With().Str("component": "foo").Logger()
//     sublogger.Info().Msg("hello world")
//     // Output: {"time":1494567715,"level":"info","message":"hello world","component":"foo"}
//
// Level logging
//
//     zerolog.SetGlobalLevel(zerolog.InfoLevel)
//
//     log.Debug().Msg("filtered out message")
//     log.Info().Msg("routed message")
//
//     if e := log.Debug(); e.Enabled() {
//         // Compute log output only if enabled.
//         value := compute()
//         e.Str("foo": value).Msg("some debug message")
//     }
//     // Output: {"level":"info","time":1494567715,"routed message"}
//
// Customize automatic field names:
//
//     log.TimestampFieldName = "t"
//     log.LevelFieldName = "p"
//     log.MessageFieldName = "m"
//
//     log.Info().Msg("hello world")
//     // Output: {"t":1494567715,"p":"info","m":"hello world"}
//
// Log with no level and message:
//
//     log.Log().Str("foo","bar").Msg("")
//     // Output: {"time":1494567715,"foo":"bar"}
//
// Add contextual fields to global Logger:
//
//     log.Logger = log.With().Str("foo", "bar").Logger()
//
// Sample logs:
//
//     sampled := log.Sample(10)
//     sampled.Info().Msg("will be logged every 10 messages")
//
package zerolog

import (
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"sync/atomic"

	"github.com/rs/zerolog/internal/json"
)

// Level defines log levels.
type Level uint8

const (
	// DebugLevel defines debug log level.
	DebugLevel Level = iota
	// InfoLevel defines info log level.
	InfoLevel
	// WarnLevel defines warn log level.
	WarnLevel
	// ErrorLevel defines error log level.
	ErrorLevel
	// FatalLevel defines fatal log level.
	FatalLevel
	// PanicLevel defines panic log level.
	PanicLevel
	// Disabled disables the logger.
	Disabled
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	}
	return ""
}

const (
	// Often samples log every 10 events.
	Often = 10
	// Sometimes samples log every 100 events.
	Sometimes = 100
	// Rarely samples log every 1000 events.
	Rarely = 1000
)

var disabledEvent = newEvent(levelWriterAdapter{ioutil.Discard}, 0, false)

// A Logger represents an active logging object that generates lines
// of JSON output to an io.Writer. Each logging operation makes a single
// call to the Writer's Write method. There is no guaranty on access
// serialization to the Writer. If your Writer is not thread safe,
// you may consider a sync wrapper.
type Logger struct {
	w       LevelWriter
	level   Level
	sample  uint32
	counter *uint32
	context []byte
}

// New creates a root logger with given output writer. If the output writer implements
// the LevelWriter interface, the WriteLevel method will be called instead of the Write
// one.
//
// Each logging operation makes a single call to the Writer's Write method. There is no
// guaranty on access serialization to the Writer. If your Writer is not thread safe,
// you may consider using sync wrapper.
func New(w io.Writer) Logger {
	if w == nil {
		w = ioutil.Discard
	}
	lw, ok := w.(LevelWriter)
	if !ok {
		lw = levelWriterAdapter{w}
	}
	return Logger{w: lw}
}

// Nop returns a disabled logger for which all operation are no-op.
func Nop() Logger {
	return New(nil).Level(Disabled)
}

// With creates a child logger with the field added to its context.
func (l Logger) With() Context {
	context := l.context
	l.context = make([]byte, 0, 500)
	if context != nil {
		l.context = append(l.context, context...)
	} else {
		// first byte of context is presence of timestamp or not
		l.context = append(l.context, 0)
	}
	return Context{l}
}

// Level creates a child logger with the minimum accepted level set to level.
func (l Logger) Level(lvl Level) Logger {
	return Logger{
		w:       l.w,
		level:   lvl,
		sample:  l.sample,
		counter: l.counter,
		context: l.context,
	}
}

// Sample returns a logger that only let one message out of every to pass thru.
func (l Logger) Sample(every int) Logger {
	if every == 0 {
		// Create a child with no sampling.
		return Logger{
			w:       l.w,
			level:   l.level,
			context: l.context,
		}
	}
	return Logger{
		w:       l.w,
		level:   l.level,
		sample:  uint32(every),
		counter: new(uint32),
		context: l.context,
	}
}

// Debug starts a new message with debug level.
//
// You must call Msg on the returned event in order to send the event.
func (l Logger) Debug() *Event {
	return l.newEvent(DebugLevel, true, nil)
}

// Info starts a new message with info level.
//
// You must call Msg on the returned event in order to send the event.
func (l Logger) Info() *Event {
	return l.newEvent(InfoLevel, true, nil)
}

// Warn starts a new message with warn level.
//
// You must call Msg on the returned event in order to send the event.
func (l Logger) Warn() *Event {
	return l.newEvent(WarnLevel, true, nil)
}

// Error starts a new message with error level.
//
// You must call Msg on the returned event in order to send the event.
func (l Logger) Error() *Event {
	return l.newEvent(ErrorLevel, true, nil)
}

// Fatal starts a new message with fatal level. The os.Exit(1) function
// is called by the Msg method.
//
// You must call Msg on the returned event in order to send the event.
func (l Logger) Fatal() *Event {
	return l.newEvent(FatalLevel, true, func(msg string) { os.Exit(1) })
}

// Panic starts a new message with panic level. The message is also sent
// to the panic function.
//
// You must call Msg on the returned event in order to send the event.
func (l Logger) Panic() *Event {
	return l.newEvent(PanicLevel, true, func(msg string) { panic(msg) })
}

// WithLevel starts a new message with level.
//
// You must call Msg on the returned event in order to send the event.
func (l Logger) WithLevel(level Level) *Event {
	switch level {
	case DebugLevel:
		return l.Debug()
	case InfoLevel:
		return l.Info()
	case WarnLevel:
		return l.Warn()
	case ErrorLevel:
		return l.Error()
	case FatalLevel:
		return l.Fatal()
	case PanicLevel:
		return l.Panic()
	case Disabled:
		return disabledEvent
	default:
		panic("zerolog: WithLevel(): invalid level: " + strconv.Itoa(int(level)))
	}
}

// Log starts a new message with no level. Setting GlobalLevel to Disabled
// will still disable events produced by this method.
//
// You must call Msg on the returned event in order to send the event.
func (l Logger) Log() *Event {
	// We use panic level with addLevelField=false to make Log passthrough all
	// levels except Disabled.
	return l.newEvent(PanicLevel, false, nil)
}

// Write implements the io.Writer interface. This is useful to set as a writer
// for the standard library log.
func (l Logger) Write(p []byte) (n int, err error) {
	n = len(p)
	if n > 0 && p[n-1] == '\n' {
		// Trim CR added by stdlog.
		p = p[0 : n-1]
	}
	l.Log().Msg(string(p))
	return
}

func (l Logger) newEvent(level Level, addLevelField bool, done func(string)) *Event {
	enabled := l.should(level)
	if !enabled {
		return disabledEvent
	}
	lvl := InfoLevel
	if addLevelField {
		lvl = level
	}
	e := newEvent(l.w, lvl, enabled)
	e.done = done
	if l.context != nil && len(l.context) > 0 && l.context[0] > 0 {
		// first byte of context is ts flag
		e.buf = json.AppendTime(json.AppendKey(e.buf, TimestampFieldName), TimestampFunc(), TimeFieldFormat)
	}
	if addLevelField {
		e.Str(LevelFieldName, level.String())
	}
	if l.sample > 0 && SampleFieldName != "" {
		e.Uint32(SampleFieldName, l.sample)
	}
	if l.context != nil && len(l.context) > 1 {
		if len(e.buf) > 1 {
			e.buf = append(e.buf, ',')
		}
		e.buf = append(e.buf, l.context[1:]...)
	}
	return e
}

// should returns true if the log event should be logged.
func (l Logger) should(lvl Level) bool {
	if lvl < l.level || lvl < globalLevel() {
		return false
	}
	if l.sample > 0 && l.counter != nil && !samplingDisabled() {
		c := atomic.AddUint32(l.counter, 1)
		return c%l.sample == 0
	}
	return true
}
