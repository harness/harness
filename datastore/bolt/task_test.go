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
