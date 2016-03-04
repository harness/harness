package matrix

import (
	"testing"
        "reflect"
	"github.com/franela/goblin"
)

func Test_noexclMatrix(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Calculate full matrix", func() {

		axis, _ := Parse(noexclMatrix)

		g.It("Should calculate permutations", func() {
			g.Assert(len(axis)).Equal(24)
		})

		g.It("Should calculate the correct permutations", func() {
	          g.Assert(len(axis)).Equal(len(perm_noexclMatrix))
                  for _, a := range axis {
                    ok := false
                    for _, c := range perm_noexclMatrix {
                      if reflect.DeepEqual(a,c) {
                        ok = true
                      }
                    }
                    g.Assert(ok).IsTrue()
                  }
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

func Test_exclMatrix(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Calculate sparse matrix", func() {

		axis, _ := Parse(exclMatrix)

		g.It("Should calculate permutations", func() {
			g.Assert(len(axis)).Equal(16)
		})

		g.It("Should calculate the correct permutations", func() {
	          g.Assert(len(axis)).Equal(len(perm_exclMatrix))
                  for _, a := range axis {
                    ok := false
                    for _, c := range perm_exclMatrix {
                      if reflect.DeepEqual(a,c) {
                        ok = true
                      }
                    }
                    g.Assert(ok).IsTrue()
                  }
		})

		g.It("Should not duplicate permutations", func() {
			set := map[string]bool{}
			for _, perm := range axis {
				set[perm.String()] = true
			}
			g.Assert(len(set)).Equal(16)
		})

		g.It("Should return nil if no matrix", func() {
			axis, err := Parse("")
			g.Assert(err == nil).IsTrue()
			g.Assert(axis == nil).IsTrue()
		})
	})
}

// example build matrix with no exclusions
var noexclMatrix = `
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
// example build matrix with exclusions
// from all permutations will remove those that contain either
//   (python_version: 3.2 AND django_version: 1.7) OR
//   (python_version: 3.2 AND django_version: 1.7.2)
var exclMatrix = `
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
exclude:
  python_version:
    - 3.2
  django_version:
    - 1.7
    - 1.7.2
`

// all correct permutations for noexclMatrix without exclusions
var perm_noexclMatrix = []Axis{
        map[string]string{ "django_version": "1.7",
                           "go_version": "go1",
                           "python_version": "3.2",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7",
                           "go_version": "go1",
                           "python_version": "3.2",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1",
                           "python_version": "3.2",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1",
                           "python_version": "3.2",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7.2",
                           "go_version": "go1",
                           "python_version": "3.2",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7.2",
                           "go_version": "go1",
                           "python_version": "3.2",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7",
                           "go_version": "go1",
                           "python_version": "3.3",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7",
                           "go_version": "go1",
                           "python_version": "3.3",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1",
                           "python_version": "3.3",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1",
                           "python_version": "3.3",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7.2",
                           "go_version": "go1",
                           "python_version": "3.3",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7.2",
                           "go_version": "go1",
                           "python_version": "3.3",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7",
                           "go_version": "go1.2",
                           "python_version": "3.2",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7",
                           "go_version": "go1.2",
                           "python_version": "3.2",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1.2",
                           "python_version": "3.2",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1.2",
                           "python_version": "3.2",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7.2",
                           "go_version": "go1.2",
                           "python_version": "3.2",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7.2",
                           "go_version": "go1.2",
                           "python_version": "3.2",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7",
                           "go_version": "go1.2",
                           "python_version": "3.3",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7",
                           "go_version": "go1.2",
                           "python_version": "3.3",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1.2",
                           "python_version": "3.3",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1.2",
                           "python_version": "3.3",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7.2",
                           "go_version": "go1.2",
                           "python_version": "3.3",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7.2",
                           "go_version": "go1.2",
                           "python_version": "3.3",
                           "redis_version": "2.8" },
}

// all correct permutations for exclMatrix (with exclusions removed)
var perm_exclMatrix = []Axis{
        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1",
                           "python_version": "3.2",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1",
                           "python_version": "3.2",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7",
                           "go_version": "go1",
                           "python_version": "3.3",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7",
                           "go_version": "go1",
                           "python_version": "3.3",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1",
                           "python_version": "3.3",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1",
                           "python_version": "3.3",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7.2",
                           "go_version": "go1",
                           "python_version": "3.3",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7.2",
                           "go_version": "go1",
                           "python_version": "3.3",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1.2",
                           "python_version": "3.2",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1.2",
                           "python_version": "3.2",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7",
                           "go_version": "go1.2",
                           "python_version": "3.3",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7",
                           "go_version": "go1.2",
                           "python_version": "3.3",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1.2",
                           "python_version": "3.3",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7.1",
                           "go_version": "go1.2",
                           "python_version": "3.3",
                           "redis_version": "2.8" },

        map[string]string{ "django_version": "1.7.2",
                           "go_version": "go1.2",
                           "python_version": "3.3",
                           "redis_version": "2.6" },

        map[string]string{ "django_version": "1.7.2",
                           "go_version": "go1.2",
                           "python_version": "3.3",
                           "redis_version": "2.8" },
}
