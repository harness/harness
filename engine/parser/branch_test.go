package parser

import (
	"testing"

	"github.com/franela/goblin"
)

func TestBranch(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Branch filter", func() {

		g.It("Should parse and match emtpy", func() {
			branch := ParseBranchString("")
			g.Assert(branch.Matches("master")).IsTrue()
		})

		g.It("Should parse and match", func() {
			branch := ParseBranchString("branches: { include: [ master, develop ] }")
			g.Assert(branch.Matches("master")).IsTrue()
		})

		g.It("Should parse and match shortand", func() {
			branch := ParseBranchString("branches: [ master, develop ]")
			g.Assert(branch.Matches("master")).IsTrue()
		})

		g.It("Should parse and match shortand string", func() {
			branch := ParseBranchString("branches: master")
			g.Assert(branch.Matches("master")).IsTrue()
		})

		g.It("Should parse and match exclude", func() {
			branch := ParseBranchString("branches: { exclude: [ master, develop ] }")
			g.Assert(branch.Matches("master")).IsFalse()
		})

		g.It("Should parse and match exclude shorthand", func() {
			branch := ParseBranchString("branches: { exclude: master }")
			g.Assert(branch.Matches("master")).IsFalse()
		})

		g.It("Should match include", func() {
			b := Branch{}
			b.Include = []string{"master"}
			g.Assert(b.Matches("master")).IsTrue()
		})

		g.It("Should match include pattern", func() {
			b := Branch{}
			b.Include = []string{"feature/*"}
			g.Assert(b.Matches("feature/foo")).IsTrue()
		})

		g.It("Should fail to match include pattern", func() {
			b := Branch{}
			b.Include = []string{"feature/*"}
			g.Assert(b.Matches("master")).IsFalse()
		})

		g.It("Should match exclude", func() {
			b := Branch{}
			b.Exclude = []string{"master"}
			g.Assert(b.Matches("master")).IsFalse()
		})

		g.It("Should match exclude pattern", func() {
			b := Branch{}
			b.Exclude = []string{"feature/*"}
			g.Assert(b.Matches("feature/foo")).IsFalse()
		})
	})
}
