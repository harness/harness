package yaml

import (
	"testing"

	"github.com/franela/goblin"
)

func TestBranch(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Branch filter", func() {

		g.It("Should parse and match emtpy", func() {
			branch := ParseBranchString("")
			g.Assert(branch.Match("master")).IsTrue()
		})

		g.It("Should parse and match", func() {
			branch := ParseBranchString("branches: { include: [ master, develop ] }")
			g.Assert(branch.Match("master")).IsTrue()
		})

		g.It("Should parse and match shortand", func() {
			branch := ParseBranchString("branches: [ master, develop ]")
			g.Assert(branch.Match("master")).IsTrue()
		})

		g.It("Should parse and match shortand string", func() {
			branch := ParseBranchString("branches: master")
			g.Assert(branch.Match("master")).IsTrue()
		})

		g.It("Should parse and match exclude", func() {
			branch := ParseBranchString("branches: { exclude: [ master, develop ] }")
			g.Assert(branch.Match("master")).IsFalse()
		})

		g.It("Should parse and match exclude shorthand", func() {
			branch := ParseBranchString("branches: { exclude: master }")
			g.Assert(branch.Match("master")).IsFalse()
		})
	})
}
