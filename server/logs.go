package server

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetLogs accepts a request to retrieve logs from the
// datastore for the given repository, build and task
// number.
//
//     GET /api/logs/:owner/:name/:number/:task
//
func GetLogs(c *gin.Context) {
	ds := ToDatastore(c)
	repo := ToRepo(c)
	build, _ := strconv.Atoi(c.Params.ByName("number"))
	task, _ := strconv.Atoi(c.Params.ByName("task"))

	logs, err := ds.GetTaskLogs(repo.FullName, build, task)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.Writer.Write(logs)
	}
}
