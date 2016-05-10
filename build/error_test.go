package build

import (
	"testing"

	"github.com/franela/goblin"
)

func TestErrors(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Error messages", func() {

		g.It("should include OOM details", func() {
			err := OomError{Name: "golang"}
			got, want := err.Error(), "golang : received oom kill"
			g.Assert(got).Equal(want)
		})

		g.It("should include Exit code", func() {
			err := ExitError{Name: "golang", Code: 255}
			got, want := err.Error(), "golang : exit code 255"
			g.Assert(got).Equal(want)
		})
	})
}
