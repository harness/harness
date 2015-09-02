package server

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin"
	"github.com/drone/drone/pkg/queue"
	common "github.com/drone/drone/pkg/types"
	"github.com/drone/drone/pkg/utils/httputil"
	"github.com/drone/drone/pkg/yaml/inject"
	"github.com/drone/drone/pkg/yaml/secure"
	// "github.com/gin-gonic/gin/binding"
)

// GetCommit accepts a request to retrieve a commit
// from the datastore for the given repository and
// commit sequence.
//
//     GET /api/repos/:owner/:name/:number
//
func GetBuild(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	num, err := strconv.Atoi(c.Params.ByName("number"))
	if err != nil {
		c.Fail(400, err)
		return
	}
	build, err := store.BuildNumber(repo, num)
	if err != nil {
		c.Fail(404, err)
		return
	}
	build.Jobs, err = store.JobList(build)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, build)
	}
}

// GetCommits accepts a request to retrieve a list
// of commits from the datastore for the given repository.
//
//     GET /api/repos/:owner/:name/builds
//
func GetBuilds(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	builds, err := store.BuildList(repo, 20, 0)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, builds)
	}
}

// GetLogs accepts a request to retrieve logs from the
// datastore for the given repository, build and task
// number.
//
//     GET /api/repos/:owner/:name/logs/:number/:task
//
func GetLogs(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	full, _ := strconv.ParseBool(c.Params.ByName("full"))
	build, _ := strconv.Atoi(c.Params.ByName("number"))
	job, _ := strconv.Atoi(c.Params.ByName("task"))

	path := fmt.Sprintf("/logs/%s/%v/%v", repo.FullName, build, job)
	r, err := store.GetBlobReader(path)
	if err != nil {
		c.Fail(404, err)
		return
	}

	defer r.Close()
	if full {
		io.Copy(c.Writer, r)
	} else {
		io.Copy(c.Writer, io.LimitReader(r, 2000000))
	}
}

// // PostBuildStatus accepts a request to create a new build
// // status. The created user status is returned in JSON
// // format if successful.
// //
// //     POST /api/repos/:owner/:name/status/:number
// //
// func PostBuildStatus(c *gin.Context) {
// 	store := ToDatastore(c)
// 	repo := ToRepo(c)
// 	num, err := strconv.Atoi(c.Params.ByName("number"))
// 	if err != nil {
// 		c.Fail(400, err)
// 		return
// 	}
// 	in := &common.Status{}
// 	if !c.BindWith(in, binding.JSON) {
// 		c.AbortWithStatus(400)
// 		return
// 	}
// 	if err := store.SetBuildStatus(repo.Name, num, in); err != nil {
// 		c.Fail(400, err)
// 	} else {
// 		c.JSON(201, in)
// 	}
// }

// RunBuild accepts a request to restart an existing build.
//
//     POST /api/builds/:owner/:name/builds/:number
//
func RunBuild(c *gin.Context) {
	remote := ToRemote(c)
	store := ToDatastore(c)
	queue_ := ToQueue(c)
	repo := ToRepo(c)
	conf := ToSettings(c)

	num, err := strconv.Atoi(c.Params.ByName("number"))
	if err != nil {
		c.Fail(400, err)
		return
	}
	build, err := store.BuildNumber(repo, num)
	if err != nil {
		c.Fail(404, err)
		return
	}
	build.Jobs, err = store.JobList(build)
	if err != nil {
		c.Fail(404, err)
		return
	}

	user, err := store.User(repo.UserID)
	if err != nil {
		c.Fail(404, err)
		return
	}

	// must not restart a running build
	if build.Status == common.StatePending || build.Status == common.StateRunning {
		c.AbortWithStatus(409)
		return
	}

	build.Status = common.StatePending
	build.Started = 0
	build.Finished = 0
	for _, job := range build.Jobs {
		job.Status = common.StatePending
		job.Started = 0
		job.Finished = 0
		job.ExitCode = 0
	}

	err = store.SetBuild(build)
	if err != nil {
		c.Fail(500, err)
		return
	}

	netrc, err := remote.Netrc(user, repo)
	if err != nil {
		c.Fail(500, err)
		return
	}

	// featch the .drone.yml file from the database
	raw, err := remote.Script(user, repo, build)
	if err != nil {
		c.Fail(404, err)
		return
	}

	// inject any private parameters into the .drone.yml
	if repo.Params != nil && len(repo.Params) != 0 {
		raw = []byte(inject.InjectSafe(string(raw), repo.Params))
	}
	encrypted, _ := secure.Parse(repo.Keys.Private, repo.Hash, string(raw))
	if encrypted != nil && len(encrypted) != 0 {
		raw = []byte(inject.InjectSafe(string(raw), encrypted))
	}

	c.JSON(202, build)

	queue_.Publish(&queue.Work{
		User:  user,
		Repo:  repo,
		Build: build,
		Keys:  repo.Keys,
		Netrc: netrc,
		Yaml:  raw,
		System: &common.System{
			Link:    httputil.GetURL(c.Request),
			Plugins: conf.Plugins,
			Globals: conf.Environment,
		},
	})
}

// KillBuild accepts a request to kill a running build.
//
//     DELETE /api/builds/:owner/:name/builds/:number
//
func KillBuild(c *gin.Context) {
	runner := ToRunner(c)
	queue := ToQueue(c)
	store := ToDatastore(c)
	repo := ToRepo(c)
	num, err := strconv.Atoi(c.Params.ByName("number"))
	if err != nil {
		c.Fail(400, err)
		return
	}
	build, err := store.BuildNumber(repo, num)
	if err != nil {
		c.Fail(404, err)
		return
	}
	build.Jobs, err = store.JobList(build)
	if err != nil {
		c.Fail(404, err)
		return
	}

	// must not restart a running build
	if build.Status != common.StatePending && build.Status != common.StateRunning {
		c.Fail(409, err)
		return
	}

	// remove from the queue if exists
	//
	// TODO(bradrydzewski) this could yield a race condition
	// because other threads may also be accessing these items.
	for _, item := range queue.Items() {
		if item.Repo.FullName == repo.FullName && item.Build.Number == build.Number {
			queue.Remove(item)
			break
		}
	}

	build.Status = common.StateKilled
	build.Finished = time.Now().Unix()
	if build.Started == 0 {
		build.Started = build.Finished
	}
	for _, job := range build.Jobs {
		if job.Status != common.StatePending && job.Status != common.StateRunning {
			continue
		}
		job.Status = common.StateKilled
		job.Started = build.Started
		job.Finished = build.Finished
	}
	err = store.SetBuild(build)
	if err != nil {
		c.Fail(500, err)
		return
	}

	for _, job := range build.Jobs {
		runner.Cancel(job)
	}
	// // get the agent from the repository so we can
	// // notify the agent to kill the build.
	// agent, err := store.BuildAgent(repo.FullName, build.Number)
	// if err != nil {
	// 	c.JSON(200, build)
	// 	return
	// }
	// url_, _ := url.Parse("http://" + agent.Addr)
	// url_.Path = fmt.Sprintf("/cancel/%s/%v", repo.FullName, build.Number)
	// resp, err := http.Post(url_.String(), "application/json", nil)
	// if err != nil {
	// 	c.Fail(500, err)
	// 	return
	// }
	// defer resp.Body.Close()

	c.JSON(200, build)
}
