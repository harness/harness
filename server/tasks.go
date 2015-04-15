package server

import (
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetTask accepts a request to retrieve a build task
// from the datastore for the given repository and
// build number.
//
//     GET /api/tasks/:owner/:name/:number/:task
//
func GetTask(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	b, _ := strconv.Atoi(c.Params.ByName("number"))
	t, _ := strconv.Atoi(c.Params.ByName("task"))

	task, err := store.Task(repo.FullName, b, t)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, task)
	}
}

// GetTasks accepts a request to retrieve a list of
// build tasks from the datastore for the given repository
// and build number.
//
//     GET /api/tasks/:owner/:name/:number
//
func GetTasks(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	num, _ := strconv.Atoi(c.Params.ByName("number"))

	tasks, err := store.TaskList(repo.FullName, num)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, tasks)
	}
}

// GetTaskLogs accepts a request to retrieve logs from the
// datastore for the given repository, build and task
// number.
//
//     GET /api/logs/:owner/:name/:number/:task
//
func GetTaskLogs(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	full, _ := strconv.ParseBool(c.Params.ByName("full"))
	build, _ := strconv.Atoi(c.Params.ByName("number"))
	task, _ := strconv.Atoi(c.Params.ByName("task"))

	r, err := store.LogReader(repo.FullName, build, task)
	if err != nil {
		c.Fail(404, err)
	} else if full {
		io.Copy(c.Writer, r)
	} else {
		io.Copy(c.Writer, io.LimitReader(r, 2000000))
	}
}
