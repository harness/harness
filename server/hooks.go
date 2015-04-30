package server

import (
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/common"
	"github.com/drone/drone/parser"
	"github.com/drone/drone/parser/inject"
	"github.com/drone/drone/parser/matrix"
	"github.com/drone/drone/queue"
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
	queue_ := ToQueue(c)

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

	repo, err := store.Repo(hook.Repo.FullName)
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

	user, err := store.User(repo.User.Login)
	if err != nil {
		log.Errorf("failure to find repo owner %s. %s", repo.User.Login, err)
		c.Fail(500, err)
		return
	}

	params, _ := store.RepoParams(repo.FullName)

	build := &common.Build{}
	build.State = common.StatePending
	build.Commit = hook.Commit
	build.PullRequest = hook.PullRequest

	// featch the .drone.yml file from the database
	raw, err := remote.Script(user, repo, build)
	if err != nil {
		log.Errorf("failure to get .drone.yml for %s. %s", repo.FullName, err)
		c.Fail(404, err)
		return
	}
	// inject any private parameters into the .drone.yml
	if params != nil && len(params) != 0 {
		raw = []byte(inject.InjectSafe(string(raw), params))
	}
	axes, err := matrix.Parse(string(raw))
	if err != nil {
		log.Errorf("failure to calculate matrix for %s. %s", repo.FullName, err)
		c.Fail(404, err)
		return
	}
	if len(axes) == 0 {
		axes = append(axes, matrix.Axis{})
	}
	for num, axis := range axes {
		build.Tasks = append(build.Tasks, &common.Task{
			Number:      num + 1,
			State:       common.StatePending,
			Environment: axis,
		})
	}
	keys, err := store.RepoKeypair(repo.FullName)
	if err != nil {
		log.Errorf("failure to fetch keypair for %s. %s", repo.FullName, err)
		c.Fail(404, err)
		return
	}

	netrc, err := remote.Netrc(user)
	if err != nil {
		c.Fail(500, err)
		return
	}

	// verify the branches can be built vs skipped
	when, _ := parser.ParseCondition(string(raw))
	if build.Commit != nil && when != nil && !when.MatchBranch(build.Commit.Ref) {
		log.Infof("ignoring hook. yaml file excludes repo and branch %s %s", repo.FullName, build.Commit.Ref)
		c.AbortWithStatus(200)
		return
	}

	err = store.SetBuild(repo.FullName, build)
	if err != nil {
		c.Fail(500, err)
		return
	}

	queue_.Publish(&queue.Work{
		User:  user,
		Repo:  repo,
		Build: build,
		Keys:  keys,
		Netrc: netrc,
		Yaml:  raw,
	})

	c.JSON(200, build)
}
