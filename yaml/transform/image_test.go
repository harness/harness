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
