package runner

import (
	"testing"

	"github.com/franela/goblin"
)

func TestSpec(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Spec file", func() {

		g.Describe("when looking up a container", func() {

			spec := Spec{}
			spec.Containers = append(spec.Containers, &Container{
				Name: "golang",
			})

			g.It("should find and return the container", func() {
				c, err := spec.lookupContainer("golang")
				g.Assert(err == nil).IsTrue("error should be nil")
				g.Assert(c).Equal(spec.Containers[0])
			})

			g.It("should return an error when not found", func() {
				c, err := spec.lookupContainer("node")
				g.Assert(err == nil).IsFalse("should return error")
				g.Assert(c == nil).IsTrue("should return nil container")
			})

		})
	})
}
