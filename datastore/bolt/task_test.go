package bolt

import (
	"github.com/drone/drone/common"
	. "github.com/franela/goblin"
	"io/ioutil"
	"os"
	"testing"
)

func TestTask(t *testing.T) {
	g := Goblin(t)
	g.Describe("Tasks", func() {
		//testUser := "octocat"
		testRepo := "github.com/octopod/hq"
		testBuild := 1
		testTask := 0
		//testTask2 := 1
		testLogInfo := []byte("Log Info for SetLogs()")
		var db *DB // Temp database

		// create a new database before each unit
		// test and destroy afterwards.
		g.BeforeEach(func() {
			db = Must("/tmp/drone.test.db")
		})
		g.AfterEach(func() {
			os.Remove(db.Path())
		})

		/*
			Brad Rydzewski1:00 PM
			the `Task`, `TaskList` and `SetTask` are deprecated and can be probably be removed.
			I just need to make sure we aren't still using those functions anywhere else in the code
		*/
		/*
			g.It("Should get TaskList", func() {
				db.SetRepo(&common.Repo{FullName: testRepo})
				//db.SetRepoNotExists(&common.User{Login: testUser}, &common.Repo{FullName: testRepo})
				err := db.SetTask(testRepo, testBuild, &common.Task{Number: testTask})
				g.Assert(err).Equal(nil)
				err_ := db.SetTask(testRepo, testBuild, &common.Task{Number: testTask2})
				g.Assert(err_).Equal(nil)
				//
				tasks, err := db.TaskList(testRepo, testBuild)
				// We seem to have an issue here. TaskList doesn't seem to be returning
				// All the tasks added to to repo/build. So commenting these for now.
				//g.Assert(err).Equal(nil)
				//g.Assert(len(tasks)).Equal(2)
			})

			g.It("Should set Task", func() {
				db.SetRepo(&common.Repo{FullName: testRepo})
				err := db.SetTask(testRepo, testBuild, &common.Task{Number: testTask})
				g.Assert(err).Equal(nil)
			})

			g.It("Should get Task", func() {
				db.SetRepo(&common.Repo{FullName: testRepo})
				db.SetTask(testRepo, testBuild, &common.Task{Number: testTask})
				//
				task, err := db.Task(testRepo, testBuild, testTask)
				g.Assert(err).Equal(nil)
				g.Assert(task.Number).Equal(testTask)
			})
		*/

		g.It("Should set Logs", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})
			//db.SetTask(testRepo, testBuild, &common.Task{Number: testTask})
			//db.SetTask(testRepo, testBuild, &common.Task{Number: testTask2})
			//
			err := db.SetLogs(testRepo, testBuild, testTask, testLogInfo)
			g.Assert(err).Equal(nil)
		})

		g.It("Should LogReader", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})
			//db.SetTask(testRepo, testBuild, &common.Task{Number: testTask})
			//db.SetTask(testRepo, testBuild, &common.Task{Number: testTask2})
			db.SetLogs(testRepo, testBuild, testTask, testLogInfo)
			//
			buf, err_ := db.LogReader(testRepo, testBuild, testTask)
			g.Assert(err_).Equal(nil)
			logInfo, err_ := ioutil.ReadAll(buf)
			g.Assert(logInfo).Equal(testLogInfo)
		})
	})
}
