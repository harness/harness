package yaml

import (
	"testing"

	"github.com/franela/goblin"
)

func TestMatrix(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Calculate matrix", func() {

		axis, _ := ParseMatrixString(fakeMatrix)

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
			axis, err := ParseMatrixString("")
			g.Assert(err == nil).IsTrue()
			g.Assert(axis == nil).IsTrue()
		})

		g.It("Should return included axis", func() {
			axis, err := ParseMatrixString(fakeMatrixInclude)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(axis)).Equal(2)
			g.Assert(axis[0]["go_version"]).Equal("1.5")
			g.Assert(axis[1]["go_version"]).Equal("1.6")
			g.Assert(axis[0]["python_version"]).Equal("3.4")
			g.Assert(axis[1]["python_version"]).Equal("3.4")
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

var fakeMatrixInclude = `
matrix:
  include:
    - go_version: 1.5
      python_version: 3.4
    - go_version: 1.6
      python_version: 3.4
`
