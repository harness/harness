package builtin

import (
	"testing"

	"github.com/drone/drone/engine/compiler/parse"
	"github.com/drone/drone/engine/runner"

	"github.com/franela/goblin"
)

func Test_escalate(t *testing.T) {
	root := parse.NewRootNode()

	g := goblin.Goblin(t)
	g.Describe("privileged transform", func() {

		g.It("should handle matches", func() {
			c := root.NewPluginNode()
			c.Container = runner.Container{Image: "plugins/docker"}
			op := NewEscalateOp([]string{"plugins/docker"})

			op.VisitContainer(c)
			g.Assert(c.Container.Privileged).IsTrue()
		})

		g.It("should handle glob matches", func() {
			c := root.NewPluginNode()
			c.Container = runner.Container{Image: "plugins/docker"}
			op := NewEscalateOp([]string{"plugins/*"})

			op.VisitContainer(c)
			g.Assert(c.Container.Privileged).IsTrue()
		})

		g.It("should handle non matches", func() {
			c := root.NewPluginNode()
			c.Container = runner.Container{Image: "plugins/git"}
			op := NewEscalateOp([]string{"plugins/docker"})

			op.VisitContainer(c)
			g.Assert(c.Container.Privileged).IsFalse()
		})

		g.It("should handle non glob matches", func() {
			c := root.NewPluginNode()
			c.Container = runner.Container{Image: "plugins/docker:develop"}
			op := NewEscalateOp([]string{"plugins/docker"})

			op.VisitContainer(c)
			g.Assert(c.Container.Privileged).IsFalse()
		})
	})
}
