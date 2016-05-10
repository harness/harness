package transform

import (
	"testing"

	"github.com/drone/drone/yaml"

	"github.com/franela/goblin"
)

func Test_env(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("environment variables", func() {

		g.It("should be copied", func() {
			envs := map[string]string{"CI": "drone"}

			c := newConfig(&yaml.Container{
				Environment: map[string]string{},
			})

			Environ(c, envs)
			g.Assert(c.Pipeline[0].Environment["CI"]).Equal("drone")
		})
	})
}
