package builtin

import (
	"testing"

	"github.com/drone/drone/engine/compiler/parse"
	"github.com/drone/drone/engine/runner"

	"github.com/franela/goblin"
)

func Test_pull(t *testing.T) {
	root := parse.NewRootNode()

	g := goblin.Goblin(t)
	g.Describe("pull image", func() {

		g.It("should be enabled for plugins", func() {
			c := root.NewPluginNode()
			c.Container = runner.Container{}
			op := NewPullOp(true)

			op.VisitContainer(c)
			g.Assert(c.Container.Pull).IsTrue()
		})

		g.It("should be disabled for plugins", func() {
			c := root.NewPluginNode()
			c.Container = runner.Container{}
			op := NewPullOp(false)

			op.VisitContainer(c)
			g.Assert(c.Container.Pull).IsFalse()
		})

		g.It("should be disabled for non-plugins", func() {
			c := root.NewShellNode()
			c.Container = runner.Container{}
			op := NewPullOp(true)

			op.VisitContainer(c)
			g.Assert(c.Container.Pull).IsFalse()
		})
	})
}
