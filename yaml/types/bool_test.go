package types

import (
	"testing"

	"github.com/franela/goblin"
	"gopkg.in/yaml.v2"
)

func TestBoolTrue(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Yaml bool type", func() {
		g.Describe("given a yaml file", func() {

			g.It("should unmarshal true", func() {
				in := []byte("true")
				out := BoolTrue{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(out.Bool()).Equal(true)
			})

			g.It("should unmarshal false", func() {
				in := []byte("false")
				out := BoolTrue{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(out.Bool()).Equal(false)
			})

			g.It("should unmarshal true when empty", func() {
				in := []byte("")
				out := BoolTrue{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(out.Bool()).Equal(true)
			})

			g.It("should throw error when invalid", func() {
				in := []byte("{ }") // string value should fail parse
				out := BoolTrue{}
				err := yaml.Unmarshal(in, &out)
				g.Assert(err != nil).IsTrue("expects error")
			})
		})
	})
}
