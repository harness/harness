package model

import (
	"testing"

	"github.com/franela/goblin"
)

func TestSecret(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Secret", func() {

		g.It("should match event", func() {
			secret := Secret{}
			secret.Events = []string{"pull_request"}
			g.Assert(secret.Match("pull_request")).IsTrue()
		})
		g.It("should not match event", func() {
			secret := Secret{}
			secret.Events = []string{"pull_request"}
			g.Assert(secret.Match("push")).IsFalse()
		})
		g.It("should match when no event filters defined", func() {
			secret := Secret{}
			g.Assert(secret.Match("pull_request")).IsTrue()
		})
		g.It("should pass validation")
		g.Describe("should fail validation", func() {
			g.It("when no image")
			g.It("when no event")
		})
	})
}
