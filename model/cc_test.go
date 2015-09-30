package model

import (
	"testing"

	"github.com/franela/goblin"
)

func TestCC(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("CC", func() {

		g.It("Should create a project", func() {

			r := &Repo{
				FullName: "foo/bar",
			}
			b := &Build{
				Status:  StatusSuccess,
				Number:  1,
				Started: 1442872675,
			}
			cc := NewCC(r, b, "http://localhost/foo/bar/1")

			g.Assert(cc.Project.Name).Equal("foo/bar")
			g.Assert(cc.Project.Activity).Equal("Sleeping")
			g.Assert(cc.Project.LastBuildStatus).Equal("Success")
			g.Assert(cc.Project.LastBuildLabel).Equal("1")
			g.Assert(cc.Project.LastBuildTime).Equal("2015-09-21T14:57:55-07:00")
			g.Assert(cc.Project.WebURL).Equal("http://localhost/foo/bar/1")
		})

		g.It("Should properly label exceptions", func() {
			r := &Repo{FullName: "foo/bar"}
			b := &Build{
				Status:  StatusError,
				Number:  1,
				Started: 1257894000,
			}
			cc := NewCC(r, b, "http://localhost/foo/bar/1")
			g.Assert(cc.Project.LastBuildStatus).Equal("Exception")
			g.Assert(cc.Project.Activity).Equal("Sleeping")
		})

		g.It("Should properly label success", func() {
			r := &Repo{FullName: "foo/bar"}
			b := &Build{
				Status:  StatusSuccess,
				Number:  1,
				Started: 1257894000,
			}
			cc := NewCC(r, b, "http://localhost/foo/bar/1")
			g.Assert(cc.Project.LastBuildStatus).Equal("Success")
			g.Assert(cc.Project.Activity).Equal("Sleeping")
		})

		g.It("Should properly label failure", func() {
			r := &Repo{FullName: "foo/bar"}
			b := &Build{
				Status:  StatusFailure,
				Number:  1,
				Started: 1257894000,
			}
			cc := NewCC(r, b, "http://localhost/foo/bar/1")
			g.Assert(cc.Project.LastBuildStatus).Equal("Failure")
			g.Assert(cc.Project.Activity).Equal("Sleeping")
		})

		g.It("Should properly label running", func() {
			r := &Repo{FullName: "foo/bar"}
			b := &Build{
				Status:  StatusRunning,
				Number:  1,
				Started: 1257894000,
			}
			cc := NewCC(r, b, "http://localhost/foo/bar/1")
			g.Assert(cc.Project.Activity).Equal("Building")
			g.Assert(cc.Project.LastBuildStatus).Equal("Unknown")
			g.Assert(cc.Project.LastBuildLabel).Equal("Unknown")
		})
	})
}
