package debug

import (
	"net/http/pprof"

	"github.com/gin-gonic/gin"
)

// IndexHandler will pass the call from /debug/pprof to pprof
func IndexHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Index(c.Writer, c.Request)
	}
}

// HeapHandler will pass the call from /debug/pprof/heap to pprof
func HeapHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Handler("heap").ServeHTTP(c.Writer, c.Request)
	}
}

// GoroutineHandler will pass the call from /debug/pprof/goroutine to pprof
func GoroutineHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Handler("goroutine").ServeHTTP(c.Writer, c.Request)
	}
}

// BlockHandler will pass the call from /debug/pprof/block to pprof
func BlockHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Handler("block").ServeHTTP(c.Writer, c.Request)
	}
}

// ThreadCreateHandler will pass the call from /debug/pprof/threadcreate to pprof
func ThreadCreateHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Handler("threadcreate").ServeHTTP(c.Writer, c.Request)
	}
}

// CmdlineHandler will pass the call from /debug/pprof/cmdline to pprof
func CmdlineHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Cmdline(c.Writer, c.Request)
	}
}

// ProfileHandler will pass the call from /debug/pprof/profile to pprof
func ProfileHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Profile(c.Writer, c.Request)
	}
}

// SymbolHandler will pass the call from /debug/pprof/symbol to pprof
func SymbolHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Symbol(c.Writer, c.Request)
	}
}

// TraceHandler will pass the call from /debug/pprof/trace to pprof
func TraceHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Trace(c.Writer, c.Request)
	}
}
