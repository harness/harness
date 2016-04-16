package runner

import (
	"testing"

	"github.com/franela/goblin"
)

func TestContainer(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Container validation", func() {

		g.It("fails with an invalid name", func() {
			c := Container{
				Image: "golang:1.5",
			}
			err := c.Validate()
			g.Assert(err != nil).IsTrue()
			g.Assert(err.Error()).Equal("Missing container name")
		})

		g.It("fails with an invalid image", func() {
			c := Container{
				Name: "container_0",
			}
			err := c.Validate()
			g.Assert(err != nil).IsTrue()
			g.Assert(err.Error()).Equal("Missing container image")
		})

		g.It("passes with valid attributes", func() {
			c := Container{
				Name:  "container_0",
				Image: "golang:1.5",
			}
			g.Assert(c.Validate() == nil).IsTrue()
		})
	})
}
