package zerolog

import (
	"context"
	"io/ioutil"
)

var disabledLogger *Logger

func init() {
	l := New(ioutil.Discard).Level(Disabled)
	disabledLogger = &l
}

type ctxKey struct{}

// WithContext returns a copy of ctx with l associated. If an instance of Logger
// is already in the context, the pointer to this logger is updated with l.
//
// For instance, to add a field to an existing logger in the context, use this
// notation:
//
//     ctx := r.Context()
//     l := zerolog.Ctx(ctx)
//     ctx = l.With().Str("foo", "bar").WithContext(ctx)
func (l Logger) WithContext(ctx context.Context) context.Context {
	if lp, ok := ctx.Value(ctxKey{}).(*Logger); ok {
		// Update existing pointer.
		*lp = l
		return ctx
	}
	if l.level == Disabled {
		// Do not store disabled logger.
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, &l)
}

// Ctx returns the Logger associated with the ctx. If no logger
// is associated, a disabled logger is returned.
func Ctx(ctx context.Context) *Logger {
	if l, ok := ctx.Value(ctxKey{}).(*Logger); ok {
		return l
	}
	return disabledLogger
}
