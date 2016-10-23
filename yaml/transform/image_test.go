package transform

import (
	"testing"

	"github.com/drone/drone/yaml"

	"github.com/franela/goblin"
)

func Test_pull(t *testing.T) {
	g := goblin.Goblin(t)
	g.Describe("pull image", func() {

		g.It("should be enabled for plugins", func() {
			c := newConfig(&yaml.Container{})

			ImagePull(c, true)
			g.Assert(c.Pipeline[0].Pull).IsTrue()
		})

		g.It("should be disabled for plugins", func() {
			c := newConfig(&yaml.Container{})

			ImagePull(c, false)
			g.Assert(c.Pipeline[0].Pull).IsFalse()
		})

		g.It("should not apply to commands", func() {
			c := newConfig(&yaml.Container{
				Commands: []string{
					"go build",
					"go test",
				},
			})

			ImagePull(c, true)
			g.Assert(c.Pipeline[0].Pull).IsFalse()
		})

		g.It("should not apply to services", func() {
			c := newConfigService(&yaml.Container{
				Image: "mysql",
			})

			ImagePull(c, true)
			g.Assert(c.Services[0].Pull).IsFalse()
		})
	})
}

func Test_escalate(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("privileged transform", func() {

		g.It("should handle matches", func() {
			c := newConfig(&yaml.Container{
				Image: "plugins/docker",
			})

			ImageEscalate(c, []string{"plugins/docker"})
			g.Assert(c.Pipeline[0].Privileged).IsTrue()
		})

		g.It("should handle glob matches", func() {
			c := newConfig(&yaml.Container{
				Image: "plugins/docker:latest",
			})

			ImageEscalate(c, []string{"plugins/docker:*"})
			g.Assert(c.Pipeline[0].Privileged).IsTrue()
		})

		g.It("should handle non matches", func() {
			c := newConfig(&yaml.Container{
				Image: "plugins/git:latest",
			})

			ImageEscalate(c, []string{"plugins/docker:*"})
			g.Assert(c.Pipeline[0].Privileged).IsFalse()
		})

		g.It("should handle non glob matches", func() {
			c := newConfig(&yaml.Container{
				Image: "plugins/docker:latest",
			})

			ImageEscalate(c, []string{"plugins/docker"})
			g.Assert(c.Pipeline[0].Privileged).IsFalse()
		})

		g.It("should not escalate plugin with commands", func() {
			c := newConfig(&yaml.Container{
				Image:    "docker",
				Commands: []string{"echo foo"},
			})

			err := ImageEscalate(c, []string{"docker"})
			g.Assert(c.Pipeline[0].Privileged).IsFalse()
			g.Assert(err.Error()).Equal("Custom commands disabled for the docker plugin")
		})
	})
}

func Test_normalize(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("normalizing", func() {

		g.Describe("images", func() {

			g.It("should append tag if empty", func() {
				c := newConfig(&yaml.Container{
					Image: "golang",
				})

				ImageTag(c)
				g.Assert(c.Pipeline[0].Image).Equal("golang:latest")
			})

			g.It("should not override existing tag", func() {
				c := newConfig(&yaml.Container{
					Image: "golang:1.5",
				})

				ImageTag(c)
				g.Assert(c.Pipeline[0].Image).Equal("golang:1.5")
			})
		})
	})
}
