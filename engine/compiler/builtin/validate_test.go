package builtin

import (
	"testing"

	"github.com/drone/drone/engine/compiler/parse"
	"github.com/drone/drone/engine/runner"

	"github.com/franela/goblin"
)

func Test_validate(t *testing.T) {
	root := parse.NewRootNode()

	g := goblin.Goblin(t)
	g.Describe("validating", func() {

		g.Describe("privileged attributes", func() {

			g.It("should not error when trusted build", func() {
				c := root.NewContainerNode()
				c.Container = runner.Container{}
				ops := NewValidateOp(true, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err == nil).IsTrue("error should be nil")
			})

			g.It("should error when privleged mode", func() {
				c := root.NewContainerNode()
				c.Container = runner.Container{}
				c.Container.Privileged = true
				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use privileged mode")
			})

			g.It("should error when dns configured", func() {
				c := root.NewContainerNode()
				c.Container = runner.Container{}
				c.Container.DNS = []string{"8.8.8.8"}
				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use custom dns")
			})

			g.It("should error when dns_search configured", func() {
				c := root.NewContainerNode()
				c.Container = runner.Container{}
				c.Container.DNSSearch = []string{"8.8.8.8"}
				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use dns_search")
			})

			g.It("should error when devices configured", func() {
				c := root.NewContainerNode()
				c.Container = runner.Container{}
				c.Container.Devices = []string{"/dev/foo"}
				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use devices")
			})

			g.It("should error when extra_hosts configured", func() {
				c := root.NewContainerNode()
				c.Container = runner.Container{}
				c.Container.ExtraHosts = []string{"1.2.3.4 foo.com"}
				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use extra_hosts")
			})

			g.It("should error when network configured", func() {
				c := root.NewContainerNode()
				c.Container = runner.Container{}
				c.Container.Network = "host"
				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to override the network")
			})

			g.It("should error when oom_kill_disabled configured", func() {
				c := root.NewContainerNode()
				c.Container = runner.Container{}
				c.Container.OomKillDisable = true
				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to disable oom_kill")
			})

			g.It("should error when volumes configured", func() {
				c := root.NewContainerNode()
				c.Container = runner.Container{}
				c.Container.Volumes = []string{"/:/tmp"}
				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use volumes")
			})

			g.It("should error when volumes_from configured", func() {
				c := root.NewContainerNode()
				c.Container = runner.Container{}
				c.Container.VolumesFrom = []string{"drone"}
				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use volumes_from")
			})
		})

		g.Describe("plugin configuration", func() {
			g.It("should error when entrypoint is configured", func() {
				c := root.NewPluginNode()
				c.Container = runner.Container{Image: "plugins/git"}
				c.Container.Entrypoint = []string{"/bin/sh"}
				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Cannot set plugin Entrypoint")
			})

			g.It("should error when command is configured", func() {
				c := root.NewPluginNode()
				c.Container = runner.Container{Image: "plugins/git"}
				c.Container.Command = []string{"cat", "/proc/1/status"}
				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Cannot set plugin Command")
			})

			g.It("should not error when empty entrypoint, command", func() {
				c := root.NewPluginNode()
				c.Container = runner.Container{Image: "plugins/git"}
				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err == nil).IsTrue("error should be nil")
			})
		})

		g.Describe("plugin whitelist", func() {

			g.It("should error when no match found", func() {
				c := root.NewPluginNode()
				c.Container = runner.Container{}
				c.Container.Image = "custom/git"

				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err != nil).IsTrue("error should be nil")
				g.Assert(err.Error()).Equal("Plugin custom/git is not in the whitelist")
			})

			g.It("should not error when match found", func() {
				c := root.NewPluginNode()
				c.Container = runner.Container{}
				c.Container.Image = "plugins/git"

				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err == nil).IsTrue("error should be nil")
			})

			g.It("should ignore build images", func() {
				c := root.NewShellNode()
				c.Container = runner.Container{}
				c.Container.Image = "google/golang"

				ops := NewValidateOp(false, []string{"plugins/*"})
				err := ops.VisitContainer(c)

				g.Assert(err == nil).IsTrue("error should be nil")
			})
		})
	})
}
