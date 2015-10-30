package cache

import (
	"testing"

	"github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func TestCache(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Cache", func() {

		var c *gin.Context
		g.BeforeEach(func() {
			c = new(gin.Context)
			ToContext(c, Default())
		})

		g.It("Should set and get an item", func() {
			Set(c, "foo", "bar")
			v, e := Get(c, "foo")
			g.Assert(v).Equal("bar")
			g.Assert(e == nil).IsTrue()
		})

		g.It("Should return nil when item not found", func() {
			v, e := Get(c, "foo")
			g.Assert(v == nil).IsTrue()
			g.Assert(e == nil).IsFalse()
		})
	})
}
