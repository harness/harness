package builtin

import (
	"testing"

	"github.com/drone/drone/engine/compiler/parse"
	"github.com/drone/drone/engine/runner"

	"github.com/franela/goblin"
)

func Test_normalize(t *testing.T) {
	root := parse.NewRootNode()

	g := goblin.Goblin(t)
	g.Describe("normalizing", func() {

		g.Describe("images", func() {

			g.It("should append tag if empty", func() {
				c := root.NewContainerNode()
				c.Container = runner.Container{Image: "golang"}
				op := NewNormalizeOp("")

				op.VisitContainer(c)
				g.Assert(c.Container.Image).Equal("golang:latest")
			})

			g.It("should not override existing tag", func() {
				c := root.NewContainerNode()
				c.Container = runner.Container{Image: "golang:1.5"}
				op := NewNormalizeOp("")

				op.VisitContainer(c)
				g.Assert(c.Container.Image).Equal("golang:1.5")
			})
		})

		g.Describe("plugins", func() {

			g.It("should prepend namespace", func() {
				c := root.NewPluginNode()
				c.Container = runner.Container{Image: "git"}
				op := NewNormalizeOp("plugins")

				op.VisitContainer(c)
				g.Assert(c.Container.Image).Equal("plugins/git:latest")
			})

			g.It("should not override existing namespace", func() {
				c := root.NewPluginNode()
				c.Container = runner.Container{Image: "index.docker.io/drone/git"}
				op := NewNormalizeOp("plugins")

				op.VisitContainer(c)
				g.Assert(c.Container.Image).Equal("index.docker.io/drone/git:latest")
			})

			g.It("should ignore shell or service types", func() {
				c := root.NewShellNode()
				c.Container = runner.Container{Image: "golang"}
				op := NewNormalizeOp("plugins")

				op.VisitContainer(c)
				g.Assert(c.Container.Image).Equal("golang:latest")
			})
		})
	})
}
