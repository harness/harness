package builtin

import (
	"testing"

	"github.com/franela/goblin"
	"github.com/drone/drone/engine/compiler/parse"
)

func Test_workspace(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("workspace", func() {

		var defaultBase = "/go"
		var defaultPath = "src/github.com/octocat/hello-world"

		g.It("should not override user paths", func() {
			var base = "/drone"
			var path = "/drone/src/github.com/octocat/hello-world"

			op := NewWorkspaceOp(defaultBase, defaultPath)
			root := parse.NewRootNode()
			root.Base = base
			root.Path = path

			op.VisitRoot(root)
			g.Assert(root.Base).Equal(base)
			g.Assert(root.Path).Equal(path)
		})

		g.It("should convert user paths to absolute", func() {
			var base = "/drone"
			var path = "src/github.com/octocat/hello-world"
			var abs = "/drone/src/github.com/octocat/hello-world"

			op := NewWorkspaceOp(defaultBase, defaultPath)
			root := parse.NewRootNode()
			root.Base = base
			root.Path = path

			op.VisitRoot(root)
			g.Assert(root.Base).Equal(base)
			g.Assert(root.Path).Equal(abs)
		})

		g.It("should set the default path", func() {
			var base = "/go"
			var path = "/go/src/github.com/octocat/hello-world"

			op := NewWorkspaceOp(defaultBase, defaultPath)
			root := parse.NewRootNode()

			op.VisitRoot(root)
			g.Assert(root.Base).Equal(base)
			g.Assert(root.Path).Equal(path)
		})

		g.It("should use workspace as working_dir", func() {
			var base = "/drone"
			var path = "/drone/src/github.com/octocat/hello-world"

			root := parse.NewRootNode()
			root.Base = base
			root.Path = path

			c := root.NewContainerNode()

			op := NewWorkspaceOp(defaultBase, defaultPath)
			op.VisitContainer(c)
			g.Assert(c.Container.WorkingDir).Equal(root.Path)
		})

		g.It("should not use workspace as working_dir for services", func() {
			var base = "/drone"
			var path = "/drone/src/github.com/octocat/hello-world"

			root := parse.NewRootNode()
			root.Base = base
			root.Path = path

			c := root.NewServiceNode()

			op := NewWorkspaceOp(defaultBase, defaultPath)
			op.VisitContainer(c)
			g.Assert(c.Container.WorkingDir).Equal("")
		})
	})
}
