package builtin

import (
	"testing"

	"github.com/drone/drone/engine/compiler/parse"
	"github.com/drone/drone/engine/runner"

	"github.com/franela/goblin"
)

func Test_args(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("plugins arguments", func() {

		g.It("should ignore non-plugin containers", func() {
			root := parse.NewRootNode()
			c := root.NewShellNode()
			c.Container = runner.Container{}
			c.Vargs = map[string]interface{}{
				"depth": 50,
			}

			ops := NewArgsOp()
			ops.VisitContainer(c)

			g.Assert(c.Container.Environment["PLUGIN_DEPTH"]).Equal("")
		})

		g.It("should include args as environment variable", func() {
			root := parse.NewRootNode()
			c := root.NewPluginNode()
			c.Container = runner.Container{}
			c.Vargs = map[string]interface{}{
				"depth": 50,
			}

			ops := NewArgsOp()
			ops.VisitContainer(c)

			g.Assert(c.Container.Environment["PLUGIN_DEPTH"]).Equal("50")
		})
	})

}
