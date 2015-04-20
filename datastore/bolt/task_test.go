package bolt

import (
	//"bytes"
	"github.com/drone/drone/common"
	. "github.com/franela/goblin"
	"os"
	"testing"
	//. "github.com/smartystreets/goconvey/convey"
)

func TestTask(t *testing.T) {
	g := Goblin(t)
	g.Describe("Tasks", func() {
		testUser := "octocat"
		testRepo := "github.com/octopod/hq"
		testBuild := 1
		testTask := 1
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

		g.It("Should set Task", func() {
			tasks := db.SetTask(testRepo, testBuild, testTask)
			g.Assert(tasks).NotEqual(nil)
		})

		g.It("Should get Task", func() {
			task, err := db.Task(testRepo, testBuild, testTask)
			g.Assert(task).Equal(testTask)
			g.Assert(err).Equal(nil)
		})

		g.It("Should get TaskList", func() {
			tasks, err := db.TaskList(testRepo, testBuild)
			g.Assert(tasks).NotEqual(nil)
			g.Assert(err).Equal(nil)
		})

		g.It("Should set Logs", func() {
			err := db.SetLogs(testRepo, testBuild, testTask, testLogInfo)
			g.Assert(err).Equal(nil)
		})

		g.It("Should LogReader", func() {
			buf, err := db.LogReader(testRepo, testBuild, testTask)
			g.Assert(buf).NotEqual(nil)
			g.Assert(err).Equal(nil)
		})
	})
}
