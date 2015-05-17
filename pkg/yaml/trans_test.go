package parser

import (
	"testing"

	common "github.com/drone/drone/pkg/types"
	"github.com/franela/goblin"
)

func Test_Transform(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Transform", func() {

		g.It("Should transform setup step", func() {
			c := &common.Config{}
			c.Build = &common.Step{}
			c.Build.Config = map[string]interface{}{}
			transformSetup(c)
			g.Assert(c.Setup != nil).IsTrue()
			g.Assert(c.Setup.Image).Equal("plugins/drone-build")
			g.Assert(c.Setup.Config).Equal(c.Build.Config)
		})

		g.It("Should transform clone step", func() {
			c := &common.Config{}
			transformClone(c)
			g.Assert(c.Clone != nil).IsTrue()
			g.Assert(c.Clone.Image).Equal("plugins/drone-git")
		})

		g.It("Should transform build", func() {
			c := &common.Config{}
			c.Build = &common.Step{}
			c.Build.Config = map[string]interface{}{}
			c.Build.Config["commands"] = []string{"echo hello"}
			transformBuild(c)
			g.Assert(len(c.Build.Config)).Equal(0)
			g.Assert(c.Build.Entrypoint[0]).Equal("/bin/bash")
			g.Assert(c.Build.Command[0]).Equal("/drone/bin/build.sh")
		})

		g.It("Should transform images", func() {
			c := &common.Config{}
			c.Setup = &common.Step{Image: "foo"}
			c.Clone = &common.Step{Image: "foo/bar"}
			c.Build = &common.Step{Image: "golang"}
			c.Publish = map[string]*common.Step{"google_compute": &common.Step{}}
			c.Deploy = map[string]*common.Step{"amazon": &common.Step{}}
			c.Notify = map[string]*common.Step{"slack": &common.Step{}}
			transformImages(c)

			g.Assert(c.Setup.Image).Equal("plugins/drone-foo")
			g.Assert(c.Clone.Image).Equal("foo/bar")
			g.Assert(c.Build.Image).Equal("golang")
			g.Assert(c.Publish["google_compute"].Image).Equal("plugins/drone-google-compute")
			g.Assert(c.Deploy["amazon"].Image).Equal("plugins/drone-amazon")
			g.Assert(c.Notify["slack"].Image).Equal("plugins/drone-slack")
		})

		g.It("Should transform docker plugin", func() {
			c := &common.Config{}
			c.Publish = map[string]*common.Step{}
			c.Publish["docker"] = &common.Step{Image: "plugins/drone-docker"}
			transformDockerPlugin(c)
			g.Assert(c.Publish["docker"].Privileged).Equal(true)
		})

		g.It("Should remove privileged flag", func() {
			c := &common.Config{}
			c.Setup = &common.Step{Privileged: true}
			c.Clone = &common.Step{Privileged: true}
			c.Build = &common.Step{Privileged: true}
			c.Compose = map[string]*common.Step{"postgres": &common.Step{Privileged: true}}
			c.Publish = map[string]*common.Step{"google": &common.Step{Privileged: true}}
			c.Deploy = map[string]*common.Step{"amazon": &common.Step{Privileged: true}}
			c.Notify = map[string]*common.Step{"slack": &common.Step{Privileged: true}}
			rmPrivileged(c)

			g.Assert(c.Setup.Privileged).Equal(false)
			g.Assert(c.Clone.Privileged).Equal(false)
			g.Assert(c.Build.Privileged).Equal(false)
			g.Assert(c.Compose["postgres"].Privileged).Equal(false)
			g.Assert(c.Publish["google"].Privileged).Equal(false)
			g.Assert(c.Deploy["amazon"].Privileged).Equal(false)
			g.Assert(c.Notify["slack"].Privileged).Equal(false)
		})

		g.It("Should not remove docker plugin privileged flag", func() {
			c := &common.Config{}
			c.Setup = &common.Step{}
			c.Clone = &common.Step{}
			c.Build = &common.Step{}
			c.Publish = map[string]*common.Step{}
			c.Publish["docker"] = &common.Step{Image: "plugins/drone-docker"}
			transformDockerPlugin(c)
			g.Assert(c.Publish["docker"].Privileged).Equal(true)
		})

		g.It("Should remove volumes", func() {
			c := &common.Config{}
			c.Setup = &common.Step{Volumes: []string{"/:/tmp"}}
			c.Clone = &common.Step{Volumes: []string{"/:/tmp"}}
			c.Build = &common.Step{Volumes: []string{"/:/tmp"}}
			c.Compose = map[string]*common.Step{"postgres": &common.Step{Volumes: []string{"/:/tmp"}}}
			c.Publish = map[string]*common.Step{"google": &common.Step{Volumes: []string{"/:/tmp"}}}
			c.Deploy = map[string]*common.Step{"amazon": &common.Step{Volumes: []string{"/:/tmp"}}}
			c.Notify = map[string]*common.Step{"slack": &common.Step{Volumes: []string{"/:/tmp"}}}
			rmVolumes(c)

			g.Assert(len(c.Setup.Volumes)).Equal(0)
			g.Assert(len(c.Clone.Volumes)).Equal(0)
			g.Assert(len(c.Build.Volumes)).Equal(0)
			g.Assert(len(c.Compose["postgres"].Volumes)).Equal(0)
			g.Assert(len(c.Publish["google"].Volumes)).Equal(0)
			g.Assert(len(c.Deploy["amazon"].Volumes)).Equal(0)
			g.Assert(len(c.Notify["slack"].Volumes)).Equal(0)
		})

		g.It("Should remove network", func() {
			c := &common.Config{}
			c.Setup = &common.Step{NetworkMode: "host"}
			c.Clone = &common.Step{NetworkMode: "host"}
			c.Build = &common.Step{NetworkMode: "host"}
			c.Compose = map[string]*common.Step{"postgres": &common.Step{NetworkMode: "host"}}
			c.Publish = map[string]*common.Step{"google": &common.Step{NetworkMode: "host"}}
			c.Deploy = map[string]*common.Step{"amazon": &common.Step{NetworkMode: "host"}}
			c.Notify = map[string]*common.Step{"slack": &common.Step{NetworkMode: "host"}}
			rmNetwork(c)

			g.Assert(c.Setup.NetworkMode).Equal("")
			g.Assert(c.Clone.NetworkMode).Equal("")
			g.Assert(c.Build.NetworkMode).Equal("")
			g.Assert(c.Compose["postgres"].NetworkMode).Equal("")
			g.Assert(c.Publish["google"].NetworkMode).Equal("")
			g.Assert(c.Deploy["amazon"].NetworkMode).Equal("")
			g.Assert(c.Notify["slack"].NetworkMode).Equal("")
		})

		g.It("Should return full qualified image name", func() {
			g.Assert(imageName("microsoft/azure")).Equal("microsoft/azure")
			g.Assert(imageName("azure")).Equal("plugins/drone-azure")
			g.Assert(imageName("azure_storage")).Equal("plugins/drone-azure-storage")
		})
	})
}
