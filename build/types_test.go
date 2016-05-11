package build

import (
	"testing"

	"github.com/franela/goblin"
)

func TestLine(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Line output", func() {
		g.It("should prefix string() with metadata", func() {
			line := Line{
				Proc: "redis",
				Time: 60,
				Pos:  1,
				Out:  "starting redis server",
			}
			g.Assert(line.String()).Equal("[redis:L1:60s] starting redis server")
		})
	})
}
