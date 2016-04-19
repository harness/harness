package middleware

import (
	"sync"

	"github.com/drone/drone/engine"
	"github.com/drone/drone/store"

	"github.com/gin-gonic/gin"
)

// Engine is a middleware function that initializes the Engine and attaches to
// the context of every http.Request.
func Engine() gin.HandlerFunc {
	var once sync.Once
	var engine_ engine.Engine

	return func(c *gin.Context) {

		once.Do(func() {
			store_ := store.FromContext(c)
			engine_ = engine.Load(store_)
		})

		engine.ToContext(c, engine_)
		c.Next()
	}
}
