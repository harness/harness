package server

import (
	"os"
	"strings"

	log "github.com/drone/drone/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin"
	"github.com/drone/drone/pkg/hash"
	"github.com/drone/drone/pkg/queue"
	common "github.com/drone/drone/pkg/types"
	"github.com/drone/drone/pkg/utils/httputil"
	"github.com/drone/drone/pkg/yaml"
	"github.com/drone/drone/pkg/yaml/matrix"
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

	repo, err := store.RepoName(hook.Repo.Owner, hook.Repo.Name)
	if err != nil {
		log.Errorf("failure to find repo %s/%s from hook. %s", hook.Repo.Owner, hook.Repo.Name, err)
		c.Fail(404, err)
		return
	}

	// get the token and verify the hook is authorized
	if c.Request.FormValue("access_token") != hash.New(repo.FullName, repo.Hash) {
		log.Errorf("invalid token sent with hook.")
		c.AbortWithStatus(403)
		return
	}

	switch {
	case repo.UserID == 0:
		log.Warnf("ignoring hook. repo %s has no owner.", repo.FullName)
		c.Writer.WriteHeader(204)
		return
	case !repo.Hooks.Push && hook.Commit != nil:
		log.Infof("ignoring hook. repo %s is disabled.", repo.FullName)
		c.Writer.WriteHeader(204)
		return
	case !repo.Hooks.PullRequest && hook.PullRequest != nil:
		log.Warnf("ignoring hook. repo %s is disabled for pull requests.", repo.FullName)
		c.Writer.WriteHeader(204)
		return
	}

	user, err := store.User(repo.UserID)
	if err != nil {
		log.Errorf("failure to find repo owner %s. %s", repo.FullName, err)
		c.Fail(500, err)
		return
	}

	build := &common.Build{}
	build.Commit = hook.Commit
	build.PullRequest = hook.PullRequest
	build.Status = common.StatePending
	build.RepoID = repo.ID

	// fetch the .drone.yml file from the database
	raw, sec, err := remote.Script(user, repo, build)
	if err != nil {
		log.Errorf("failure to get .drone.yml for %s. %s", repo.FullName, err)
		c.Fail(404, err)
		return
	}

	axes, err := matrix.Parse(string(raw))
	if err != nil {
		log.Errorf("failure to calculate matrix for %s. %s", repo.FullName, err)
		c.Fail(400, err)
		return
	}
	if len(axes) == 0 {
		axes = append(axes, matrix.Axis{})
	}
	for num, axis := range axes {
		build.Jobs = append(build.Jobs, &common.Job{
			BuildID:     build.ID,
			Number:      num + 1,
			Status:      common.StatePending,
			Environment: axis,
		})
	}

	netrc, err := remote.Netrc(user, repo)
	if err != nil {
		log.Errorf("failure to generate netrc for %s. %s", repo.FullName, err)
		c.Fail(500, err)
		return
	}

	// verify the branches can be built vs skipped
	when, _ := parser.ParseCondition(string(raw))
	if build.PullRequest != nil && when != nil && !when.MatchBranch(build.Commit.Branch) {
		log.Infof("ignoring hook. yaml file excludes repo and branch %s %s", repo.FullName, build.Commit.Branch)
		c.AbortWithStatus(200)
		return
	}

	err = store.AddBuild(build)
	if err != nil {
		log.Errorf("failure to save commit for %s. %s", repo.FullName, err)
		c.Fail(500, err)
		return
	}

	c.JSON(200, build)

	err = remote.Status(user, repo, build)
	if err != nil {
		log.Errorf("error setting commit status for %s/%d", repo.FullName, build.Number)
	}

	queue_.Publish(&queue.Work{
		User:   user,
		Repo:   repo,
		Build:  build,
		Keys:   repo.Keys,
		Netrc:  netrc,
		Config: raw,
		Secret: sec,
		System: &common.System{
			Link:    httputil.GetURL(c.Request),
			Plugins: strings.Split(os.Getenv("PLUGIN_FILTER"), " "),
			Globals: strings.Split(os.Getenv("PLUGIN_PARAMS"), " "),
		},
	})
}
