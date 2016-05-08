package types

import (
	"testing"

	"github.com/franela/goblin"
	"gopkg.in/yaml.v2"
)

func TestMapEqualSlice(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Yaml map equal slice", func() {

		g.It("should unmarshal a map", func() {
			in := []byte("foo: bar")
			out := MapEqualSlice{}
			err := yaml.Unmarshal(in, &out)
			if err != nil {
				g.Fail(err)
			}
			g.Assert(len(out.Map())).Equal(1)
			g.Assert(out.Map()["foo"]).Equal("bar")
		})

		g.It("should unmarshal a map equal slice", func() {
			in := []byte("[ foo=bar ]")
			out := MapEqualSlice{}
			err := yaml.Unmarshal(in, &out)
			if err != nil {
				g.Fail(err)
			}
			g.Assert(len(out.parts)).Equal(1)
			g.Assert(out.parts["foo"]).Equal("bar")
		})

		g.It("should throw error when invalid map equal slice", func() {
			in := []byte("foo") // string value should fail parse
			out := MapEqualSlice{}
			err := yaml.Unmarshal(in, &out)
			g.Assert(err != nil).IsTrue("expects error")
		})
	})
}
