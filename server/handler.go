package server

import (
	"github.com/drone/drone/bus"
	"github.com/drone/drone/cache"
	"github.com/drone/drone/queue"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/store"
	"github.com/drone/drone/stream"
	"github.com/drone/drone/version"

	"github.com/gin-gonic/gin"
)

// HandlerCache returns a HandlerFunc that passes a Cache to the Context.
func HandlerCache(v cache.Cache) gin.HandlerFunc {
	return func(c *gin.Context) {
		cache.ToContext(c, v)
	}
}

// HandlerBus returns a HandlerFunc that passes a Bus to the Context.
func HandlerBus(v bus.Bus) gin.HandlerFunc {
	return func(c *gin.Context) {
		bus.ToContext(c, v)
	}
}

// HandlerStore returns a HandlerFunc that passes a Store to the Context.
func HandlerStore(v store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		store.ToContext(c, v)
	}
}

// HandlerQueue returns a HandlerFunc that passes a Queue to the Context.
func HandlerQueue(v queue.Queue) gin.HandlerFunc {
	return func(c *gin.Context) {
		queue.ToContext(c, v)
	}
}

// HandlerStream returns a HandlerFunc that passes a Stream to the Context.
func HandlerStream(v stream.Stream) gin.HandlerFunc {
	return func(c *gin.Context) {
		stream.ToContext(c, v)
	}
}

// HandlerRemote returns a HandlerFunc that passes a Remote to the Context.
func HandlerRemote(v remote.Remote) gin.HandlerFunc {
	return func(c *gin.Context) {
		remote.ToContext(c, v)
	}
}

// HandlerConfig returns a HandlerFunc that passes server Config to the Context.
func HandlerConfig(v *Config) gin.HandlerFunc {
	const k = "config"
	return func(c *gin.Context) {
		c.Set(k, v)
	}
}

// HandlerVersion returns a HandlerFunc that writes the Version information to
// the http.Response as a the X-Drone-Version header.
func HandlerVersion() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Drone-Version", version.Version)
	}
}

// HandlerAgent returns a HandlerFunc that passes an Agent token to the Context.
func HandlerAgent(v string) gin.HandlerFunc {
	const k = "agent"
	return func(c *gin.Context) {
		c.Set(k, v)
	}
}
