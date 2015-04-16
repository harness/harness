package matrix

import (
	"testing"

	"github.com/franela/goblin"
)

func Test_Matrix(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Calculate matrix", func() {

		m := map[string][]string{}
		m["go_version"] = []string{"go1", "go1.2"}
		m["python_version"] = []string{"3.2", "3.3"}
		m["django_version"] = []string{"1.7", "1.7.1", "1.7.2"}
		m["redis_version"] = []string{"2.6", "2.8"}
		axis := Calc(m)

		g.It("Should calculate permutations", func() {
			g.Assert(len(axis)).Equal(24)
		})

		g.It("Should not duplicate permutations", func() {
			set := map[string]bool{}
			for _, perm := range axis {
				set[perm.String()] = true
			}
			g.Assert(len(set)).Equal(24)
		})
	})
}
