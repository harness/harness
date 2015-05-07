package builtin

import (
	"bytes"
	"github.com/drone/drone/common"
	. "github.com/franela/goblin"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestTask(t *testing.T) {
	g := Goblin(t)
	g.Describe("Tasks", func() {

		testRepo := "octopod/hq"
		testBuild := 1
		testTask := 0

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
			//err := db.SetLogs(testRepo, testBuild, testTask, testLogInfo)
			err := db.SetLogs(testRepo, testBuild, testTask, io.Reader(bytes.NewBuffer(testLogInfo)))
			g.Assert(err).Equal(nil)
		})

		g.It("Should get logs", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})
			//db.SetLogs(testRepo, testBuild, testTask, testLogInfo)
			db.SetLogs(testRepo, testBuild, testTask, io.Reader(bytes.NewBuffer(testLogInfo)))
			buf, err := db.LogReader(testRepo, testBuild, testTask)
			g.Assert(err).Equal(nil)
			logInfo, err := ioutil.ReadAll(buf)
			g.Assert(logInfo).Equal(testLogInfo)
		})
	})
}
