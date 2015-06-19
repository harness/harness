package builtin

import (
	"testing"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/franela/goblin"
	"github.com/drone/drone/pkg/types"
)

func TestBuildstore(t *testing.T) {
	db := mustConnectTest()
	bs := NewJobstore(db)
	cs := NewCommitstore(db)
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Jobstore", func() {

		// before each test we purge the package table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM jobs")
			db.Exec("DELETE FROM commits")
		})

		g.It("Should Set a job", func() {
			job := &types.Job{
				BuildID:  1,
				Status:   "pending",
				ExitCode: 0,
				Number:   1,
			}
			err1 := bs.AddJob(job)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(job.ID != 0).IsTrue()

			job.Status = "started"
			err2 := bs.SetJob(job)
			g.Assert(err2 == nil).IsTrue()

			getjob, err3 := bs.Job(job.ID)
			g.Assert(err3 == nil).IsTrue()
			g.Assert(getjob.Status).Equal(job.Status)
		})

		g.It("Should Get a Job by ID", func() {
			job := &types.Job{
				BuildID:     1,
				Status:      "pending",
				ExitCode:    1,
				Number:      1,
				Environment: map[string]string{"foo": "bar"},
			}
			err1 := bs.AddJob(job)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(job.ID != 0).IsTrue()

			getjob, err2 := bs.Job(job.ID)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(getjob.ID).Equal(job.ID)
			g.Assert(getjob.Status).Equal(job.Status)
			g.Assert(getjob.ExitCode).Equal(job.ExitCode)
			g.Assert(getjob.Environment).Equal(job.Environment)
			g.Assert(getjob.Environment["foo"]).Equal("bar")
		})

		g.It("Should Get a Job by Number", func() {
			job := &types.Job{
				BuildID:  1,
				Status:   "pending",
				ExitCode: 1,
				Number:   1,
			}
			err1 := bs.AddJob(job)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(job.ID != 0).IsTrue()

			getjob, err2 := bs.JobNumber(&types.Commit{ID: 1}, 1)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(getjob.ID).Equal(job.ID)
			g.Assert(getjob.Status).Equal(job.Status)
		})

		g.It("Should Get a List of Jobs by Commit", func() {

			//In order for buid to be populated,
			//The AddCommit command will insert builds
			//if the Commit.Builds array is populated
			//Add Commit.
			commit := types.Commit{
				RepoID: 1,
				State:  types.StateSuccess,
				Ref:    "refs/heads/master",
				Sha:    "14710626f22791619d3b7e9ccf58b10374e5b76d",
				Builds: []*types.Job{
					&types.Job{
						BuildID:  1,
						Status:   "success",
						ExitCode: 0,
						Number:   1,
					},
					&types.Job{
						BuildID:  3,
						Status:   "error",
						ExitCode: 1,
						Number:   2,
					},
					&types.Job{
						BuildID:  5,
						Status:   "pending",
						ExitCode: 0,
						Number:   3,
					},
				},
			}
			//
			err1 := cs.AddCommit(&commit)
			g.Assert(err1 == nil).IsTrue()
			getjobs, err2 := bs.JobList(&commit)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(len(getjobs)).Equal(3)
			g.Assert(getjobs[0].Number).Equal(1)
			g.Assert(getjobs[0].Status).Equal(types.StateSuccess)
		})
	})
}
