package yaml

import (
	"testing"

	"github.com/franela/goblin"
	"gopkg.in/yaml.v2"
)

func TestBuild(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Build", func() {
		g.Describe("given a yaml file", func() {

			g.It("should unmarshal", func() {
				in := []byte(".")
				out := Build{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(out.Context).Equal(".")
			})

			g.It("should unmarshal shorthand", func() {
				in := []byte("{ context: ., dockerfile: Dockerfile }")
				out := Build{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(out.Context).Equal(".")
				g.Assert(out.Dockerfile).Equal("Dockerfile")
			})
		})
	})
}
