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
		g.It("should pass validation", func() {
			secret := Secret{}
			secret.Name = "secretname"
			secret.Value = "secretvalue"
			err := secret.Validate()
			g.Assert(err).Equal(nil)
		})
		g.Describe("should fail validation", func() {
			g.It("when no name", func() {
				secret := Secret{}
				secret.Value = "secretvalue"
				err := secret.Validate()
				g.Assert(err != nil).IsTrue()
			})
			g.It("when no value", func() {
				secret := Secret{}
				secret.Name = "secretname"
				err := secret.Validate()
				g.Assert(err != nil).IsTrue()
			})
		})
	})
}
