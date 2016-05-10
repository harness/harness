package transform

import (
	"testing"

	"github.com/drone/drone/yaml"

	"github.com/franela/goblin"
)

func Test_command(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Command genration", func() {

		g.It("should ignore plugin steps", func() {
			c := newConfig(&yaml.Container{
				Commands: []string{
					"go build",
					"go test",
				},
				Vargs: map[string]interface{}{
					"depth": 50,
				},
			})

			CommandTransform(c)
			g.Assert(len(c.Pipeline[0].Entrypoint)).Equal(0)
			g.Assert(len(c.Pipeline[0].Command)).Equal(0)
			g.Assert(c.Pipeline[0].Environment["DRONE_SCRIPT"]).Equal("")
		})

		g.It("should set entrypoint, command and environment variables", func() {
			c := newConfig(&yaml.Container{
				Commands: []string{
					"go build",
					"go test",
				},
			})

			CommandTransform(c)
			g.Assert(c.Pipeline[0].Entrypoint).Equal([]string{"/bin/sh", "-c"})
			g.Assert(c.Pipeline[0].Command).Equal([]string{"echo $DRONE_SCRIPT | base64 -d | /bin/sh -e"})
			g.Assert(c.Pipeline[0].Environment["DRONE_SCRIPT"] != "").IsTrue()
		})
	})
}
