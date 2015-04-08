package server

import (
	"strings"

	"github.com/drone/drone/common"
	// "github.com/bradrydzewski/drone/worker"
	"github.com/gin-gonic/gin"
)

// PostHook accepts a post-commit hook and parses the payload
// in order to trigger a build.
//
//     GET /api/hook
//
func PostHook(c *gin.Context) {
	remote := ToRemote(c)
	store := ToDatastore(c)

	hook, err := remote.Hook(c.Request)
	if err != nil {
		c.Fail(400, err)
		return
	}
	if hook == nil {
		c.Writer.WriteHeader(200)
		return
	}
	if hook.Repo == nil {
		c.Writer.WriteHeader(400)
		return
	}

	// a build may be skipped if the text [CI SKIP]
	// is found inside the commit message
	if hook.Commit != nil && strings.Contains(hook.Commit.Message, "[CI SKIP]") {
		c.Writer.WriteHeader(204)
		return
	}

	repo, err := store.GetRepo(hook.Repo.FullName)
	if err != nil {
		c.Fail(404, err)
		return
	}

	if repo.Disabled || repo.User == nil || (repo.DisablePR && hook.PullRequest != nil) {
		c.Writer.WriteHeader(204)
		return
	}

	user, err := store.GetUser(repo.User.Login)
	if err != nil {
		c.Fail(500, err)
		return
	}

	build := &common.Build{}
	build.State = common.StatePending
	build.Commit = hook.Commit
	build.PullRequest = hook.PullRequest

	// featch the .drone.yml file from the database
	_, err = remote.Script(user, repo, build)
	if err != nil {
		c.Fail(404, err)
		return
	}

	err = store.InsertBuild(repo.FullName, build)
	if err != nil {
		c.Fail(500, err)
		return
	}

	// w := worker.Work{
	// 	User: user,
	// 	Repo: repo,
	// 	Build: build,
	// }

	// verify the branches can be built vs skipped
	// s, _ := script.ParseBuild(string(yml))
	// if len(hook.PullRequest) == 0 && !s.MatchBranch(hook.Branch) {
	// 	w.WriteHeader(http.StatusOK)
	// 	return
	// }

	c.JSON(200, build)
}
