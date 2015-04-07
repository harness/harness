package bolt

import (
	"testing"

	. "github.com/franela/goblin"
)

func TestRepo(t *testing.T) {
	g := Goblin(t)
	g.Describe("Repos", func() {

		g.It("Should find by name")
		g.It("Should find params")
		g.It("Should find keys")
		g.It("Should delete")
		g.It("Should insert")
		g.It("Should not insert if exists")
		g.It("Should insert params")
		g.It("Should update params")
		g.It("Should insert keys")
		g.It("Should update keys")
	})
}
