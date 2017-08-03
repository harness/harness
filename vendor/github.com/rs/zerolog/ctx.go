package zerolog

import (
	"context"
	"io/ioutil"
)

var disabledLogger = New(ioutil.Discard).Level(Disabled)

type ctxKey struct{}

// WithContext returns a copy of ctx with l associated.
func (l Logger) WithContext(ctx context.Context) context.Context {
	if lp, ok := ctx.Value(ctxKey{}).(*Logger); ok {
		// Update existing pointer.
		*lp = l
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, &l)
}

// Ctx returns the Logger associated with the ctx. If no logger
// is associated, a disabled logger is returned.
func Ctx(ctx context.Context) Logger {
	if l, ok := ctx.Value(ctxKey{}).(*Logger); ok {
		return *l
	}
	return disabledLogger
}
