package log

import (
	"fmt"
	"io"
	"os"
	"sync"
)

const (
	LOG_EMERG = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

var mu sync.Mutex

// the default Log priority
var priority int = LOG_DEBUG

// the default Log output destination
var output io.Writer = os.Stdout

// the log prefix
var prefix string

// the log suffix
var suffix string = "\n"

// SetPriority sets the default log level.
func SetPriority(level int) {
	mu.Lock()
	defer mu.Unlock()
	priority = level
}

// SetOutput sets the output destination.
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	output = w
}

// SetPrefix sets the prefix for the log message.
func SetPrefix(pre string) {
	mu.Lock()
	defer mu.Unlock()
	prefix = pre
}

// SetSuffix sets the suffix for the log message.
func SetSuffix(suf string) {
	mu.Lock()
	defer mu.Unlock()
	suffix = suf
}

func Write(out string, level int) {
	mu.Lock()
	defer mu.Unlock()

	// append the prefix and suffix
	out = prefix + out + suffix

	if priority >= level {
		output.Write([]byte(out))
	}
}

func Debug(out string) {
	Write(out, LOG_DEBUG)
}

func Debugf(format string, a ...interface{}) {
	Debug(fmt.Sprintf(format, a...))
}

func Info(out string) {
	Write(out, LOG_INFO)
}

func Infof(format string, a ...interface{}) {
	Info(fmt.Sprintf(format, a...))
}

func Err(out string) {
	Write(out, LOG_ERR)
}

func Errf(format string, a ...interface{}) {
	Err(fmt.Sprintf(format, a...))
}

func Notice(out string) {
	Write(out, LOG_NOTICE)
}

func Noticef(format string, a ...interface{}) {
	Notice(fmt.Sprintf(format, a...))
}
