package transform

import (
	"testing"

	"github.com/franela/goblin"

	"github.com/drone/drone/yaml"
)

func TestWorkspace(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("workspace", func() {

		defaultBase := "/go"
		defaultPath := "src/github.com/octocat/hello-world"

		g.It("should not override user paths", func() {
			base := "/drone"
			path := "/drone/src/github.com/octocat/hello-world"

			conf := &yaml.Config{
				Workspace: &yaml.Workspace{
					Base: base,
					Path: path,
				},
			}

			WorkspaceTransform(conf, defaultBase, defaultPath)
			g.Assert(conf.Workspace.Base).Equal(base)
			g.Assert(conf.Workspace.Path).Equal(path)
		})

		g.It("should convert user paths to absolute", func() {
			base := "/drone"
			path := "src/github.com/octocat/hello-world"
			abs := "/drone/src/github.com/octocat/hello-world"

			conf := &yaml.Config{
				Workspace: &yaml.Workspace{
					Base: base,
					Path: path,
				},
			}

			WorkspaceTransform(conf, defaultBase, defaultPath)
			g.Assert(conf.Workspace.Base).Equal(base)
			g.Assert(conf.Workspace.Path).Equal(abs)
		})

		g.It("should set the default path", func() {
			var base = "/go"
			var path = "/go/src/github.com/octocat/hello-world"

			conf := &yaml.Config{}

			WorkspaceTransform(conf, defaultBase, defaultPath)
			g.Assert(conf.Workspace.Base).Equal(base)
			g.Assert(conf.Workspace.Path).Equal(path)
		})

		g.It("should use workspace as working_dir", func() {
			var base = "/drone"
			var path = "/drone/src/github.com/octocat/hello-world"

			conf := &yaml.Config{
				Workspace: &yaml.Workspace{
					Base: base,
					Path: path,
				},
				Pipeline: []*yaml.Container{
					{},
				},
			}

			WorkspaceTransform(conf, defaultBase, defaultPath)
			g.Assert(conf.Pipeline[0].WorkingDir).Equal(path)
		})

		g.It("should not use workspace as working_dir for services", func() {
			var base = "/drone"
			var path = "/drone/src/github.com/octocat/hello-world"

			conf := &yaml.Config{
				Workspace: &yaml.Workspace{
					Base: base,
					Path: path,
				},
				Services: []*yaml.Container{
					{},
				},
			}

			WorkspaceTransform(conf, defaultBase, defaultPath)
			g.Assert(conf.Services[0].WorkingDir).Equal("")
		})
	})
}
