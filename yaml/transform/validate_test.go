package transform

import (
	"testing"

	"github.com/drone/drone/yaml"

	"github.com/franela/goblin"
)

func Test_validate(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("validating", func() {

		g.Describe("privileged attributes", func() {

			g.It("should not error when trusted build", func() {
				c := newConfig(&yaml.Container{Privileged: true})
				err := Check(c, true)

				g.Assert(err == nil).IsTrue("error should be nil")
			})

			g.It("should error when privleged mode", func() {
				c := newConfig(&yaml.Container{
					Privileged: true,
				})
				err := Check(c, false)
				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use privileged mode")
			})

			g.It("should error when privleged service container", func() {
				c := newConfigService(&yaml.Container{
					Privileged: true,
				})
				err := Check(c, false)
				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use privileged mode")
			})

			g.It("should error when dns configured", func() {
				c := newConfig(&yaml.Container{
					DNS: []string{"8.8.8.8"},
				})
				err := Check(c, false)
				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use custom dns")
			})

			g.It("should error when dns_search configured", func() {
				c := newConfig(&yaml.Container{
					DNSSearch: []string{"8.8.8.8"},
				})
				err := Check(c, false)
				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use dns_search")
			})

			g.It("should error when devices configured", func() {
				c := newConfig(&yaml.Container{
					Devices: []string{"/dev/foo"},
				})
				err := Check(c, false)
				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use devices")
			})

			g.It("should error when extra_hosts configured", func() {
				c := newConfig(&yaml.Container{
					ExtraHosts: []string{"1.2.3.4 foo.com"},
				})
				err := Check(c, false)
				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use extra_hosts")
			})

			g.It("should error when network configured", func() {
				c := newConfig(&yaml.Container{
					Network: "host",
				})
				err := Check(c, false)
				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to override the network")
			})

			g.It("should error when oom_kill_disabled configured", func() {
				c := newConfig(&yaml.Container{
					OomKillDisable: true,
				})
				err := Check(c, false)
				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to disable oom_kill")
			})

			g.It("should error when volumes configured", func() {
				c := newConfig(&yaml.Container{
					Volumes: []string{"/:/tmp"},
				})
				err := Check(c, false)
				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use volumes")
			})

			g.It("should error when volumes_from configured", func() {
				c := newConfig(&yaml.Container{
					VolumesFrom: []string{"drone"},
				})
				err := Check(c, false)
				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Insufficient privileges to use volumes_from")
			})
		})

		g.Describe("plugin configuration", func() {
			g.It("should error when entrypoint is configured", func() {
				c := newConfig(&yaml.Container{
					Entrypoint: []string{"/bin/sh"},
				})
				err := Check(c, false)
				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Cannot set plugin Entrypoint")
			})

			g.It("should error when command is configured", func() {
				c := newConfig(&yaml.Container{
					Command: []string{"cat", "/proc/1/status"},
				})
				err := Check(c, false)
				g.Assert(err != nil).IsTrue("error should not be nil")
				g.Assert(err.Error()).Equal("Cannot set plugin Command")
			})

			g.It("should not error when empty entrypoint, command", func() {
				c := newConfig(&yaml.Container{})
				err := Check(c, false)
				g.Assert(err == nil).IsTrue("error should be nil")
			})
		})
	})
}

func newConfig(container *yaml.Container) *yaml.Config {
	return &yaml.Config{
		Pipeline: []*yaml.Container{container},
	}
}

func newConfigService(container *yaml.Container) *yaml.Config {
	return &yaml.Config{
		Services: []*yaml.Container{container},
	}
}
