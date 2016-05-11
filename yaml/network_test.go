package yaml

import (
	"testing"

	"github.com/franela/goblin"
	"gopkg.in/yaml.v2"
)

func TestNetworks(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Networks", func() {
		g.Describe("given a yaml file", func() {

			g.It("should unmarshal", func() {
				in := []byte("foo: { driver: overlay }")
				out := networkList{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(len(out.networks)).Equal(1)
				g.Assert(out.networks[0].Name).Equal("foo")
				g.Assert(out.networks[0].Driver).Equal("overlay")
			})

			g.It("should unmarshal named", func() {
				in := []byte("foo: { name: bar }")
				out := networkList{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(len(out.networks)).Equal(1)
				g.Assert(out.networks[0].Name).Equal("bar")
			})

			g.It("should unmarshal and use default driver", func() {
				in := []byte("foo: { name: bar }")
				out := networkList{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(len(out.networks)).Equal(1)
				g.Assert(out.networks[0].Driver).Equal("bridge")
			})
		})
	})
}
