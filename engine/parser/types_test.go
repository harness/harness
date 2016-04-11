package parser

import (
	"testing"

	"github.com/franela/goblin"
	"gopkg.in/yaml.v2"
)

func TestTypes(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Yaml types", func() {
		g.Describe("given a yaml file", func() {

			g.It("should unmarshal a string", func() {
				in := []byte("foo")
				out := stringOrSlice{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(len(out.parts)).Equal(1)
				g.Assert(out.parts[0]).Equal("foo")
			})

			g.It("should unmarshal a string slice", func() {
				in := []byte("[ foo ]")
				out := stringOrSlice{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(len(out.parts)).Equal(1)
				g.Assert(out.parts[0]).Equal("foo")
			})

			g.It("should throw error when invalid string slice", func() {
				in := []byte("{ }") // string value should fail parse
				out := stringOrSlice{}
				err := yaml.Unmarshal(in, &out)
				g.Assert(err != nil).IsTrue("expects error")
			})
		})
	})
}
