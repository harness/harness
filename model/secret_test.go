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
		g.It("should match image patterns", func() {
			secret := Secret{}
			secret.Images = []string{"golang:*"}
			g.Assert(secret.MatchImage("golang:1.4.2")).IsTrue()
		})
		g.It("should match any image", func() {
			secret := Secret{}
			secret.Images = []string{"*"}
			g.Assert(secret.MatchImage("custom/golang")).IsTrue()
		})
		g.It("should match any event", func() {
			secret := Secret{}
			secret.Events = []string{"*"}
			g.Assert(secret.MatchEvent("pull_request")).IsTrue()
		})
		g.It("should not match image", func() {
			secret := Secret{}
			secret.Images = []string{"golang"}
			g.Assert(secret.MatchImage("node")).IsFalse()
		})
		g.It("should not match image substring", func() {
			secret := Secret{}
			secret.Images = []string{"golang"}

			// image is only authorized for golang, not golang:1.4.2
			g.Assert(secret.MatchImage("golang:1.4.2")).IsFalse()
		})
		g.It("should not match empty image", func() {
			secret := Secret{}
			secret.Images = []string{}
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
