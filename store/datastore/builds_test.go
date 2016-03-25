package datastore

import (
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
)

func TestBuilds(t *testing.T) {
	db := openTest()
	defer db.Close()

	s := From(db)
	g := goblin.Goblin(t)
	g.Describe("Builds", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM builds")
			db.Exec("DELETE FROM jobs")
		})

		g.It("Should Post a Build", func() {
			build := model.Build{
				RepoID: 1,
				Status: model.StatusSuccess,
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			err := s.CreateBuild(&build, []*model.Job{}...)
			g.Assert(err == nil).IsTrue()
			g.Assert(build.ID != 0).IsTrue()
			g.Assert(build.Number).Equal(1)
			g.Assert(build.Commit).Equal("85f8c029b902ed9400bc600bac301a0aadb144ac")
		})

		g.It("Should Put a Build", func() {
			build := model.Build{
				RepoID: 1,
				Number: 5,
				Status: model.StatusSuccess,
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			s.CreateBuild(&build, []*model.Job{}...)
			build.Status = model.StatusRunning
			err1 := s.UpdateBuild(&build)
			getbuild, err2 := s.GetBuild(build.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(build.ID).Equal(getbuild.ID)
			g.Assert(build.RepoID).Equal(getbuild.RepoID)
			g.Assert(build.Status).Equal(getbuild.Status)
			g.Assert(build.Number).Equal(getbuild.Number)
		})

		g.It("Should Get a Build", func() {
			build := model.Build{
				RepoID: 1,
				Status: model.StatusSuccess,
			}
			s.CreateBuild(&build, []*model.Job{}...)
			getbuild, err := s.GetBuild(build.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(build.ID).Equal(getbuild.ID)
			g.Assert(build.RepoID).Equal(getbuild.RepoID)
			g.Assert(build.Status).Equal(getbuild.Status)
		})

		g.It("Should Get a Build by Number", func() {
			build1 := &model.Build{
				RepoID: 1,
				Status: model.StatusPending,
			}
			build2 := &model.Build{
				RepoID: 1,
				Status: model.StatusPending,
			}
			err1 := s.CreateBuild(build1, []*model.Job{}...)
			err2 := s.CreateBuild(build2, []*model.Job{}...)
			getbuild, err3 := s.GetBuildNumber(&model.Repo{ID: 1}, build2.Number)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(build2.ID).Equal(getbuild.ID)
			g.Assert(build2.RepoID).Equal(getbuild.RepoID)
			g.Assert(build2.Number).Equal(getbuild.Number)
		})

		g.It("Should Get a Build by Ref", func() {
			build1 := &model.Build{
				RepoID: 1,
				Status: model.StatusPending,
				Ref:    "refs/pull/5",
			}
			build2 := &model.Build{
				RepoID: 1,
				Status: model.StatusPending,
				Ref:    "refs/pull/6",
			}
			err1 := s.CreateBuild(build1, []*model.Job{}...)
			err2 := s.CreateBuild(build2, []*model.Job{}...)
			getbuild, err3 := s.GetBuildRef(&model.Repo{ID: 1}, "refs/pull/6")
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(build2.ID).Equal(getbuild.ID)
			g.Assert(build2.RepoID).Equal(getbuild.RepoID)
			g.Assert(build2.Number).Equal(getbuild.Number)
			g.Assert(build2.Ref).Equal(getbuild.Ref)
		})

		g.It("Should Get a Build by Ref", func() {
			build1 := &model.Build{
				RepoID: 1,
				Status: model.StatusPending,
				Ref:    "refs/pull/5",
			}
			build2 := &model.Build{
				RepoID: 1,
				Status: model.StatusPending,
				Ref:    "refs/pull/6",
			}
			err1 := s.CreateBuild(build1, []*model.Job{}...)
			err2 := s.CreateBuild(build2, []*model.Job{}...)
			getbuild, err3 := s.GetBuildRef(&model.Repo{ID: 1}, "refs/pull/6")
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(build2.ID).Equal(getbuild.ID)
			g.Assert(build2.RepoID).Equal(getbuild.RepoID)
			g.Assert(build2.Number).Equal(getbuild.Number)
			g.Assert(build2.Ref).Equal(getbuild.Ref)
		})

		g.It("Should Get a Build by Commit", func() {
			build1 := &model.Build{
				RepoID: 1,
				Status: model.StatusPending,
				Branch: "master",
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			build2 := &model.Build{
				RepoID: 1,
				Status: model.StatusPending,
				Branch: "dev",
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144aa",
			}
			err1 := s.CreateBuild(build1, []*model.Job{}...)
			err2 := s.CreateBuild(build2, []*model.Job{}...)
			getbuild, err3 := s.GetBuildCommit(&model.Repo{ID: 1}, build2.Commit, build2.Branch)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(build2.ID).Equal(getbuild.ID)
			g.Assert(build2.RepoID).Equal(getbuild.RepoID)
			g.Assert(build2.Number).Equal(getbuild.Number)
			g.Assert(build2.Commit).Equal(getbuild.Commit)
			g.Assert(build2.Branch).Equal(getbuild.Branch)
		})

		g.It("Should Get the last Build", func() {
			build1 := &model.Build{
				RepoID: 1,
				Status: model.StatusFailure,
				Branch: "master",
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144ac",
				Event:  model.EventPush,
			}
			build2 := &model.Build{
				RepoID: 1,
				Status: model.StatusSuccess,
				Branch: "master",
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144aa",
				Event:  model.EventPush,
			}
			err1 := s.CreateBuild(build1, []*model.Job{}...)
			err2 := s.CreateBuild(build2, []*model.Job{}...)
			getbuild, err3 := s.GetBuildLast(&model.Repo{ID: 1}, build2.Branch)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(build2.ID).Equal(getbuild.ID)
			g.Assert(build2.RepoID).Equal(getbuild.RepoID)
			g.Assert(build2.Number).Equal(getbuild.Number)
			g.Assert(build2.Status).Equal(getbuild.Status)
			g.Assert(build2.Branch).Equal(getbuild.Branch)
			g.Assert(build2.Commit).Equal(getbuild.Commit)
		})

		g.It("Should Get the last Build Before Build N", func() {
			build1 := &model.Build{
				RepoID: 1,
				Status: model.StatusFailure,
				Branch: "master",
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			build2 := &model.Build{
				RepoID: 1,
				Status: model.StatusSuccess,
				Branch: "master",
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144aa",
			}
			build3 := &model.Build{
				RepoID: 1,
				Status: model.StatusRunning,
				Branch: "master",
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144aa",
			}
			err1 := s.CreateBuild(build1, []*model.Job{}...)
			err2 := s.CreateBuild(build2, []*model.Job{}...)
			err3 := s.CreateBuild(build3, []*model.Job{}...)
			getbuild, err4 := s.GetBuildLastBefore(&model.Repo{ID: 1}, build3.Branch, build3.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(err4 == nil).IsTrue()
			g.Assert(build2.ID).Equal(getbuild.ID)
			g.Assert(build2.RepoID).Equal(getbuild.RepoID)
			g.Assert(build2.Number).Equal(getbuild.Number)
			g.Assert(build2.Status).Equal(getbuild.Status)
			g.Assert(build2.Branch).Equal(getbuild.Branch)
			g.Assert(build2.Commit).Equal(getbuild.Commit)
		})

		g.It("Should get recent Builds", func() {
			build1 := &model.Build{
				RepoID: 1,
				Status: model.StatusFailure,
			}
			build2 := &model.Build{
				RepoID: 1,
				Status: model.StatusSuccess,
			}
			s.CreateBuild(build1, []*model.Job{}...)
			s.CreateBuild(build2, []*model.Job{}...)
			builds, err := s.GetBuildList(&model.Repo{ID: 1})
			g.Assert(err == nil).IsTrue()
			g.Assert(len(builds)).Equal(2)
			g.Assert(builds[0].ID).Equal(build2.ID)
			g.Assert(builds[0].RepoID).Equal(build2.RepoID)
			g.Assert(builds[0].Status).Equal(build2.Status)
		})
	})
}
