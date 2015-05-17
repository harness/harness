package server

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/drone/drone/common"
	"github.com/drone/drone/pkg/yaml/inject"
	"github.com/drone/drone/queue"
	"github.com/gin-gonic/gin"
	// "github.com/gin-gonic/gin/binding"
)

// GetCommit accepts a request to retrieve a commit
// from the datastore for the given repository and
// commit sequence.
//
//     GET /api/repos/:owner/:name/:number
//
func GetCommit(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	num, err := strconv.Atoi(c.Params.ByName("number"))
	if err != nil {
		c.Fail(400, err)
		return
	}
	commit, err := store.CommitSeq(repo, num)
	if err != nil {
		c.Fail(404, err)
	}
	commit.Builds, err = store.BuildList(commit)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, commit)
	}
}

// GetCommits accepts a request to retrieve a list
// of commits from the datastore for the given repository.
//
//     GET /api/repos/:owner/:name/builds
//
func GetCommits(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	commits, err := store.CommitList(repo, 20, 0)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, commits)
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
	commit, _ := strconv.Atoi(c.Params.ByName("number"))
	build, _ := strconv.Atoi(c.Params.ByName("task"))

	path := fmt.Sprintf("/logs/%s/%v/%v", repo.FullName, commit, build)
	r, err := store.GetBlobReader(path)
	if err != nil {
		c.Fail(404, err)
	} else if full {
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
	num, err := strconv.Atoi(c.Params.ByName("number"))
	if err != nil {
		c.Fail(400, err)
		return
	}
	commit, err := store.CommitSeq(repo, num)
	if err != nil {
		c.Fail(404, err)
		return
	}
	commit.Builds, err = store.BuildList(commit)
	if err != nil {
		c.Fail(404, err)
		return
	}

	keys := &common.Keypair{
		Public:  repo.PublicKey,
		Private: repo.PrivateKey,
	}

	user, err := store.User(repo.UserID)
	if err != nil {
		c.Fail(404, err)
		return
	}

	// must not restart a running build
	if commit.State == common.StatePending || commit.State == common.StateRunning {
		c.AbortWithStatus(409)
		return
	}

	commit.State = common.StatePending
	commit.Started = 0
	commit.Finished = 0
	for _, build := range commit.Builds {
		build.State = common.StatePending
		build.Started = 0
		build.Finished = 0
		build.Duration = 0
		build.ExitCode = 0
	}

	err = store.SetCommit(commit)
	if err != nil {
		c.Fail(500, err)
		return
	}

	netrc, err := remote.Netrc(user)
	if err != nil {
		c.Fail(500, err)
		return
	}

	// featch the .drone.yml file from the database
	raw, err := remote.Script(user, repo, commit)
	if err != nil {
		c.Fail(404, err)
		return
	}

	// inject any private parameters into the .drone.yml
	if repo.Params != nil && len(repo.Params) != 0 {
		raw = []byte(inject.InjectSafe(string(raw), repo.Params))
	}

	c.JSON(202, commit)

	queue_.Publish(&queue.Work{
		User:   user,
		Repo:   repo,
		Commit: commit,
		Keys:   keys,
		Netrc:  netrc,
		Yaml:   raw,
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
	commit, err := store.CommitSeq(repo, num)
	if err != nil {
		c.Fail(404, err)
		return
	}
	commit.Builds, err = store.BuildList(commit)
	if err != nil {
		c.Fail(404, err)
		return
	}

	// must not restart a running build
	if commit.State != common.StatePending && commit.State != common.StateRunning {
		c.Fail(409, err)
		return
	}

	// remove from the queue if exists
	//
	// TODO(bradrydzewski) this could yield a race condition
	// because other threads may also be accessing these items.
	for _, item := range queue.Items() {
		if item.Repo.FullName == repo.FullName && item.Commit.Sequence == commit.Sequence {
			queue.Remove(item)
			break
		}
	}

	commit.State = common.StateKilled
	commit.Finished = time.Now().Unix()
	if commit.Started == 0 {
		commit.Started = commit.Finished
	}
	for _, build := range commit.Builds {
		if build.State != common.StatePending && build.State != common.StateRunning {
			continue
		}
		build.State = common.StateKilled
		build.Started = commit.Started
		build.Finished = commit.Finished
		build.Duration = commit.Finished - commit.Started
	}
	err = store.SetCommit(commit)
	if err != nil {
		c.Fail(500, err)
		return
	}

	for _, build := range commit.Builds {
		runner.Cancel(build)
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

	c.JSON(200, commit)
}
