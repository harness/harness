package model

import (
	"testing"
	"time"

	"github.com/franela/goblin"
)

func TestCC(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("CC", func() {

		g.It("Should create a project", func() {
			now := time.Now().Unix()
			now_fmt := time.Unix(now, 0).Format(time.RFC3339)
			r := &Repo{
				FullName: "foo/bar",
			}
			b := &Build{
				Status:  StatusSuccess,
				Number:  1,
				Started: now,
			}
			cc := NewCC(r, b, "http://localhost/foo/bar/1")

			g.Assert(cc.Project.Name).Equal("foo/bar")
			g.Assert(cc.Project.Activity).Equal("Sleeping")
			g.Assert(cc.Project.LastBuildStatus).Equal("Success")
			g.Assert(cc.Project.LastBuildLabel).Equal("1")
			g.Assert(cc.Project.LastBuildTime).Equal(now_fmt)
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
