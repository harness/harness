package cache

import (
	"testing"

	"github.com/franela/goblin"
)

func TestCache(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Cache", func() {

		g.BeforeEach(func() {
			cache.Purge()
		})

		g.It("should set and get item", func() {
			set("foo", "bar", 1000)
			val, expired := get("foo")
			g.Assert(val).Equal("bar")
			g.Assert(expired).Equal(false)
		})

		g.It("should return nil when item not found", func() {
			val, expired := get("foo")
			g.Assert(val == nil).IsTrue()
			g.Assert(expired).Equal(false)
		})

		g.It("should get expired item and purge", func() {
			set("foo", "bar", -900)
			val, expired := get("foo")
			g.Assert(val).Equal("bar")
			g.Assert(expired).Equal(true)
			val, _ = get("foo")
			g.Assert(val == nil).IsTrue()
		})
	})
}
