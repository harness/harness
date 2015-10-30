package raven

import (
	"reflect"
	"regexp"
)

var errorMsgPattern = regexp.MustCompile(`\A(\w+): (.+)\z`)

func NewException(err error, stacktrace *Stacktrace) *Exception {
	msg := err.Error()
	ex := &Exception{
		Stacktrace: stacktrace,
		Value:      msg,
		Type:       reflect.TypeOf(err).String(),
	}
	if m := errorMsgPattern.FindStringSubmatch(msg); m != nil {
		ex.Module, ex.Value = m[1], m[2]
	}
	return ex
}

// http://sentry.readthedocs.org/en/latest/developer/interfaces/index.html#sentry.interfaces.Exception
type Exception struct {
	// Required
	Value string `json:"value"`

	// Optional
	Type       string      `json:"type,omitempty"`
	Module     string      `json:"module,omitempty"`
	Stacktrace *Stacktrace `json:"stacktrace,omitempty"`
}

func (e *Exception) Class() string { return "exception" }

func (e *Exception) Culprit() string {
	if e.Stacktrace == nil {
		return ""
	}
	return e.Stacktrace.Culprit()
}
