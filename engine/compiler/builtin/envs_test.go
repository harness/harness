package builtin

import (
	"testing"

	"github.com/drone/drone/engine/compiler/parse"
	"github.com/drone/drone/engine/runner"

	"github.com/franela/goblin"
)

func Test_env(t *testing.T) {
	root := parse.NewRootNode()

	g := goblin.Goblin(t)
	g.Describe("environment variables", func() {

		g.It("should be copied", func() {
			envs := map[string]string{"CI": "drone"}

			c := root.NewContainerNode()
			c.Container = runner.Container{}
			op := NewEnvOp(envs)

			op.VisitContainer(c)
			g.Assert(c.Container.Environment["CI"]).Equal("drone")
		})

		g.It("should include http proxy variables", func() {
			httpProxy = "foo"
			httpsProxy = "bar"
			noProxy = "baz"

			c := root.NewContainerNode()
			c.Container = runner.Container{}
			op := NewEnvOp(map[string]string{})

			op.VisitContainer(c)
			g.Assert(c.Container.Environment["HTTP_PROXY"]).Equal("foo")
			g.Assert(c.Container.Environment["HTTPS_PROXY"]).Equal("bar")
			g.Assert(c.Container.Environment["NO_PROXY"]).Equal("baz")
		})

	})
}
