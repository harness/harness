package yaml

import (
	"testing"

	"github.com/franela/goblin"
	"gopkg.in/yaml.v2"
)

func TestVolumes(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Volumes", func() {
		g.Describe("given a yaml file", func() {

			g.It("should unmarshal", func() {
				in := []byte("foo: { driver: blockbridge }")
				out := volumeList{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(len(out.volumes)).Equal(1)
				g.Assert(out.volumes[0].Name).Equal("foo")
				g.Assert(out.volumes[0].Driver).Equal("blockbridge")
			})

			g.It("should unmarshal named", func() {
				in := []byte("foo: { name: bar }")
				out := volumeList{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(len(out.volumes)).Equal(1)
				g.Assert(out.volumes[0].Name).Equal("bar")
			})

			g.It("should unmarshal and use default driver", func() {
				in := []byte("foo: { name: bar }")
				out := volumeList{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(len(out.volumes)).Equal(1)
				g.Assert(out.volumes[0].Driver).Equal("local")
			})
		})
	})
}
