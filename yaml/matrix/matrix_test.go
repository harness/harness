package matrix

import (
	"testing"

	"github.com/franela/goblin"
)

func Test_Matrix(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Calculate matrix", func() {

		axis, _ := Parse(fakeMatrix)

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

		g.It("Should return nil if no matrix", func() {
			axis, err := Parse("")
			g.Assert(err == nil).IsTrue()
			g.Assert(axis == nil).IsTrue()
		})
	})
}

var fakeMatrix = `
matrix:
  go_version:
    - go1
    - go1.2
  python_version:
    - 3.2
    - 3.3
  django_version:
    - 1.7
    - 1.7.1
    - 1.7.2
  redis_version:
    - 2.6
    - 2.8
`
