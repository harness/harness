package bolt

import (
	"bytes"
	"os"
	"testing"
	//. "github.com/franela/goblin"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTask(t *testing.T) {
	var testUser string = "octocat"
	var testRepo string = "github.com/octopod/hq"
	var testBuild int = 1
	var testTask int = 1
	var testLogInfo []byte = "Log Info for UpsertTaskLogs"
	var db *DB //-- Temp database

	//-- We create a temp db.
	db = Must("/tmp/drone.test.db")

	//--
	Convey("Should GetTask", t, func() {
		task, err := db.GetTask(testRepo, testBuild, testTask)
		So(task, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})

	//--
	Convey("Should GetTaskLogs", t, func() {
		buf, err := db.GetTaskLogs(testRepo, testBuild, testTask)
		So(buf, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})

	//--
	Convey("Should GetTaskList", t, func() {
		tasks := db.GetTaskList(testRepo, testBuild)
		So(tasks, ShouldNotBeNil)
	})

	//--
	Convey("Should UpsertTask", t, func() {
		tasks := db.UpsertTask(testRepo, testBuild, testTask)
		So(tasks, ShouldNotBeNil)
	})

	//--
	Convey("Should UpsertTaskLogs", t, func() {
		tasks := db.UpsertTaskLogs(testRepo, testBuild, testLogInfo)
		So(tasks, ShouldNotBeNil)
		//So(err, ShouldBeNil)
	})

	//-- Delete the temp db at the end.
	os.Remove(db.Path())
}


