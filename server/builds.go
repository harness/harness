package server

import (
	"io"
	"strconv"

	"github.com/drone/drone/common"
	"github.com/drone/drone/queue"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// GetBuild accepts a request to retrieve a build
// from the datastore for the given repository and
// build number.
//
//     GET /api/builds/:owner/:name/:number
//
func GetBuild(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	num, err := strconv.Atoi(c.Params.ByName("number"))
	if err != nil {
		c.Fail(400, err)
		return
	}
	build, err := store.Build(repo.FullName, num)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, build)
	}
}

// GetBuilds accepts a request to retrieve a list
// of builds from the datastore for the given repository.
//
//     GET /api/builds/:owner/:name
//
func GetBuilds(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	builds, err := store.BuildList(repo.FullName)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, builds)
	}
}

// GetBuildLogs accepts a request to retrieve logs from the
// datastore for the given repository, build and task
// number.
//
//     GET /api/repos/:owner/:name/logs/:number/:task
//
func GetBuildLogs(c *gin.Context) {
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

// PostBuildStatus accepts a request to create a new build
// status. The created user status is returned in JSON
// format if successful.
//
//     POST /api/repos/:owner/:name/status/:number
//
func PostBuildStatus(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	num, err := strconv.Atoi(c.Params.ByName("number"))
	if err != nil {
		c.Fail(400, err)
		return
	}
	in := &common.Status{}
	if !c.BindWith(in, binding.JSON) {
		c.AbortWithStatus(400)
		return
	}
	if err := store.SetBuildStatus(repo.Name, num, in); err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(201, in)
	}
}

// RunBuild accepts a request to restart an existing build.
//
//     POST /api/builds/:owner/:name/builds/:number
//
func RunBuild(c *gin.Context) {
	store := ToDatastore(c)
	queue_ := ToQueue(c)
	repo := ToRepo(c)
	num, err := strconv.Atoi(c.Params.ByName("number"))
	if err != nil {
		c.Fail(400, err)
		return
	}
	build, err := store.Build(repo.FullName, num)
	if err != nil {
		c.Fail(404, err)
		return
	}

	keys, err := store.RepoKeypair(repo.FullName)
	if err != nil {
		c.Fail(404, err)
		return
	}

	user, err := store.User(repo.User.Login)
	if err != nil {
		c.Fail(404, err)
		return
	}

	// must not restart a running build
	if build.State == common.StatePending || build.State == common.StateRunning {
		c.Fail(409, err)
		return
	}

	build.State = common.StatePending
	build.Started = 0
	build.Finished = 0
	build.Duration = 0
	build.Statuses = []*common.Status{}
	for _, task := range build.Tasks {
		task.State = common.StatePending
		task.Started = 0
		task.Finished = 0
		task.ExitCode = 0
	}

	err = store.SetBuild(repo.FullName, build)
	if err != nil {
		c.Fail(500, err)
		return
	}

	// params, _ := store.RepoParams(repo.FullName)
	// if params != nil && len(params) != 0 {
	// 	raw = []byte(inject.InjectSafe(string(raw), params))
	// }

	queue_.Publish(&queue.Work{
		User:  user,
		Repo:  repo,
		Build: build,
		Keys:  keys,
		Netrc: &common.Netrc{}, //TODO create netrc
		Yaml:  nil,             // TODO fetch yaml
	})

	c.JSON(202, build)
}

// KillBuild accepts a request to kill a running build.
//
//     DELETE /api/builds/:owner/:name/builds/:number
//
func KillBuild(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	num, err := strconv.Atoi(c.Params.ByName("number"))
	if err != nil {
		c.Fail(400, err)
		return
	}
	build, err := store.Build(repo.FullName, num)
	if err != nil {
		c.Fail(404, err)
		return
	}

	// must not restart a running build
	if build.State != common.StatePending && build.State != common.StateRunning {
		c.Fail(409, err)
		return
	}

	// TODO tell queue to cancel build

	c.Writer.WriteHeader(202)
}
