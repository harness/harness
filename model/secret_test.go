package model

import (
	"testing"

	"github.com/franela/goblin"
)

func TestSecret(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Secret", func() {

		g.It("should match image", func() {
			secret := Secret{}
			secret.Images = []string{"golang"}
			g.Assert(secret.MatchImage("golang")).IsTrue()
		})
		g.It("should match event", func() {
			secret := Secret{}
			secret.Events = []string{"pull_request"}
			g.Assert(secret.MatchEvent("pull_request")).IsTrue()
		})
		g.It("should not match image", func() {
			secret := Secret{}
			secret.Images = []string{"golang"}
			g.Assert(secret.MatchImage("node")).IsFalse()
		})
		g.It("should not match event", func() {
			secret := Secret{}
			secret.Events = []string{"pull_request"}
			g.Assert(secret.MatchEvent("push")).IsFalse()
		})
		g.It("should pass validation")
		g.Describe("should fail validation", func() {
			g.It("when no image")
			g.It("when no event")
		})
	})
}
