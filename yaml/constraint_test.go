package yaml

import (
	"testing"

	"github.com/franela/goblin"
	"gopkg.in/yaml.v2"
)

func TestConstraint(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Constraint", func() {

		g.It("Should parse and match emtpy", func() {
			c := parseConstraint("")
			g.Assert(c.Match("master")).IsTrue()
		})

		g.It("Should parse and match", func() {
			c := parseConstraint("{ include: [ master, develop ] }")
			g.Assert(c.Include[0]).Equal("master")
			g.Assert(c.Include[1]).Equal("develop")
			g.Assert(c.Match("master")).IsTrue()
		})

		g.It("Should parse and match shortand", func() {
			c := parseConstraint("[ master, develop ]")
			g.Assert(c.Include[0]).Equal("master")
			g.Assert(c.Include[1]).Equal("develop")
			g.Assert(c.Match("master")).IsTrue()
		})

		g.It("Should parse and match shortand string", func() {
			c := parseConstraint("master")
			g.Assert(c.Include[0]).Equal("master")
			g.Assert(c.Match("master")).IsTrue()
		})

		g.It("Should parse and match exclude", func() {
			c := parseConstraint("{ exclude: [ master, develop ] }")
			g.Assert(c.Exclude[0]).Equal("master")
			g.Assert(c.Exclude[1]).Equal("develop")
			g.Assert(c.Match("master")).IsFalse()
		})

		g.It("Should parse and match exclude shorthand", func() {
			c := parseConstraint("{ exclude: master }")
			g.Assert(c.Exclude[0]).Equal("master")
			g.Assert(c.Match("master")).IsFalse()
		})

		g.It("Should match include", func() {
			b := Constraint{}
			b.Include = []string{"master"}
			g.Assert(b.Match("master")).IsTrue()
		})

		g.It("Should match include pattern", func() {
			b := Constraint{}
			b.Include = []string{"feature/*"}
			g.Assert(b.Match("feature/foo")).IsTrue()
		})

		g.It("Should fail to match include pattern", func() {
			b := Constraint{}
			b.Include = []string{"feature/*"}
			g.Assert(b.Match("master")).IsFalse()
		})

		g.It("Should match exclude", func() {
			b := Constraint{}
			b.Exclude = []string{"master"}
			g.Assert(b.Match("master")).IsFalse()
		})

		g.It("Should match exclude pattern", func() {
			b := Constraint{}
			b.Exclude = []string{"feature/*"}
			g.Assert(b.Match("feature/foo")).IsFalse()
		})

		g.It("Should match when eclude patterns mismatch", func() {
			b := Constraint{}
			b.Exclude = []string{"foo"}
			g.Assert(b.Match("bar")).IsTrue()
		})
	})
}

func TestConstraintMap(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Constraint Map", func() {
		g.It("Should parse and match emtpy", func() {
			p := map[string]string{"golang": "1.5", "redis": "3.2"}
			c := parseConstraintMap("")
			g.Assert(c.Match(p)).IsTrue()
		})

		g.It("Should parse and match", func() {
			p := map[string]string{"golang": "1.5", "redis": "3.2"}
			c := parseConstraintMap("{ include: { golang: 1.5 } }")
			g.Assert(c.Include["golang"]).Equal("1.5")
			g.Assert(c.Match(p)).IsTrue()
		})

		g.It("Should parse and match shortand", func() {
			p := map[string]string{"golang": "1.5", "redis": "3.2"}
			c := parseConstraintMap("{ golang: 1.5 }")
			g.Assert(c.Include["golang"]).Equal("1.5")
			g.Assert(c.Match(p)).IsTrue()
		})

		g.It("Should parse and match exclude", func() {
			p := map[string]string{"golang": "1.5", "redis": "3.2"}
			c := parseConstraintMap("{ exclude: { golang: 1.5 } }")
			g.Assert(c.Exclude["golang"]).Equal("1.5")
			g.Assert(c.Match(p)).IsFalse()
		})

		g.It("Should parse and mismatch exclude", func() {
			p := map[string]string{"golang": "1.5", "redis": "3.2"}
			c := parseConstraintMap("{ exclude: { golang: 1.5, redis: 2.8 } }")
			g.Assert(c.Exclude["golang"]).Equal("1.5")
			g.Assert(c.Exclude["redis"]).Equal("2.8")
			g.Assert(c.Match(p)).IsTrue()
		})
	})
}

func parseConstraint(s string) *Constraint {
	c := &Constraint{}
	yaml.Unmarshal([]byte(s), c)
	return c
}

func parseConstraintMap(s string) *ConstraintMap {
	c := &ConstraintMap{}
	yaml.Unmarshal([]byte(s), c)
	return c
}
