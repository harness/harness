package builtin

import (
	"testing"

	"github.com/drone/drone/engine/compiler/parse"
	"github.com/drone/drone/engine/runner"

	"github.com/franela/goblin"
)

func Test_shell(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("shell containers", func() {

		g.It("should ignore plugin steps", func() {
			root := parse.NewRootNode()
			c := root.NewPluginNode()
			c.Container = runner.Container{}
			ops := NewShellOp(Linux_adm64)
			ops.VisitContainer(c)

			g.Assert(len(c.Container.Entrypoint)).Equal(0)
			g.Assert(len(c.Container.Command)).Equal(0)
			g.Assert(c.Container.Environment["CI_CMDS"]).Equal("")
		})

		g.It("should set entrypoint, command and environment variables", func() {
			root := parse.NewRootNode()
			root.Base = "/go"
			root.Path = "/go/src/github.com/octocat/hello-world"

			c := root.NewShellNode()
			c.Commands = []string{"go build"}
			ops := NewShellOp(Linux_adm64)
			ops.VisitContainer(c)

			g.Assert(c.Container.Entrypoint).Equal([]string{"/bin/sh", "-c"})
			g.Assert(c.Container.Command).Equal([]string{"echo $CI_CMDS | base64 -d | /bin/sh -e"})
			g.Assert(c.Container.Environment["CI_CMDS"] != "").IsTrue()
		})
	})
}
