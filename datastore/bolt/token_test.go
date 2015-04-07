package bolt

import (
	"testing"

	. "github.com/franela/goblin"
)

func TestToken(t *testing.T) {
	g := Goblin(t)
	g.Describe("Tokens", func() {

		g.It("Should find by sha")
		g.It("Should list for user")
		g.It("Should delete")
		g.It("Should insert")
		g.It("Should not insert if exists")
	})
}
