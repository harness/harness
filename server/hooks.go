package server

import (
	"strings"

	log "github.com/Sirupsen/logrus"
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
		log.Errorf("failure to parse hook. %s", err)
		c.Fail(400, err)
		return
	}
	if hook == nil {
		c.Writer.WriteHeader(200)
		return
	}
	if hook.Repo == nil {
		log.Errorf("failure to ascertain repo from hook.")
		c.Writer.WriteHeader(400)
		return
	}

	// a build may be skipped if the text [CI SKIP]
	// is found inside the commit message
	if hook.Commit != nil && strings.Contains(hook.Commit.Message, "[CI SKIP]") {
		log.Infof("ignoring hook. [ci skip] found for %s")
		c.Writer.WriteHeader(204)
		return
	}

	repo, err := store.GetRepo(hook.Repo.FullName)
	if err != nil {
		log.Errorf("failure to find repo %s from hook. %s", hook.Repo.FullName, err)
		c.Fail(404, err)
		return
	}

	switch {
	case repo.Disabled:
		log.Infof("ignoring hook. repo %s is disabled.", repo.FullName)
		c.Writer.WriteHeader(204)
		return
	case repo.User == nil:
		log.Warnf("ignoring hook. repo %s has no owner.", repo.FullName)
		c.Writer.WriteHeader(204)
		return
	case repo.DisablePR && hook.PullRequest != nil:
		log.Warnf("ignoring hook. repo %s is disabled for pull requests.", repo.FullName)
		c.Writer.WriteHeader(204)
		return
	}

	user, err := store.GetUser(repo.User.Login)
	if err != nil {
		log.Errorf("failure to find repo owner %s. %s", repo.User.Login, err)
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
		log.Errorf("failure to get .drone.yml for %s. %s", repo.FullName, err)
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
