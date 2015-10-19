package model

import (
	"testing"

	"github.com/drone/drone/shared/database"
	"github.com/franela/goblin"
)

func TestJob(t *testing.T) {
	db := database.OpenTest()
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Job", func() {

		// before each test we purge the package table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM jobs")
			db.Exec("DELETE FROM builds")
		})

		g.It("Should Set a job", func() {
			job := &Job{
				BuildID:  1,
				Status:   "pending",
				ExitCode: 0,
				Number:   1,
			}
			err1 := InsertJob(db, job)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(job.ID != 0).IsTrue()

			job.Status = "started"
			err2 := UpdateJob(db, job)
			g.Assert(err2 == nil).IsTrue()

			getjob, err3 := GetJob(db, job.ID)
			g.Assert(err3 == nil).IsTrue()
			g.Assert(getjob.Status).Equal(job.Status)
		})

		g.It("Should Get a Job by ID", func() {
			job := &Job{
				BuildID:     1,
				Status:      "pending",
				ExitCode:    1,
				Number:      1,
				Environment: map[string]string{"foo": "bar"},
			}
			err1 := InsertJob(db, job)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(job.ID != 0).IsTrue()

			getjob, err2 := GetJob(db, job.ID)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(getjob.ID).Equal(job.ID)
			g.Assert(getjob.Status).Equal(job.Status)
			g.Assert(getjob.ExitCode).Equal(job.ExitCode)
			g.Assert(getjob.Environment).Equal(job.Environment)
			g.Assert(getjob.Environment["foo"]).Equal("bar")
		})

		g.It("Should Get a Job by Number", func() {
			job := &Job{
				BuildID:  1,
				Status:   "pending",
				ExitCode: 1,
				Number:   1,
			}
			err1 := InsertJob(db, job)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(job.ID != 0).IsTrue()

			getjob, err2 := GetJobNumber(db, &Build{ID: 1}, 1)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(getjob.ID).Equal(job.ID)
			g.Assert(getjob.Status).Equal(job.Status)
		})

		g.It("Should Get a List of Jobs by Commit", func() {

			build := Build{
				RepoID: 1,
				Status: StatusSuccess,
			}
			jobs := []*Job{
				&Job{
					BuildID:  1,
					Status:   "success",
					ExitCode: 0,
					Number:   1,
				},
				&Job{
					BuildID:  3,
					Status:   "error",
					ExitCode: 1,
					Number:   2,
				},
				&Job{
					BuildID:  5,
					Status:   "pending",
					ExitCode: 0,
					Number:   3,
				},
			}
			//
			err1 := CreateBuild(db, &build, jobs...)
			g.Assert(err1 == nil).IsTrue()
			getjobs, err2 := GetJobList(db, &build)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(len(getjobs)).Equal(3)
			g.Assert(getjobs[0].Number).Equal(1)
			g.Assert(getjobs[0].Status).Equal(StatusSuccess)
		})
	})
}
