package server

import (
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/pkg/queue"
	common "github.com/drone/drone/pkg/types"
	"github.com/drone/drone/pkg/yaml"
	"github.com/drone/drone/pkg/yaml/inject"
	"github.com/drone/drone/pkg/yaml/matrix"
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
	sess := ToSession(c)

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

	// get the token and verify the hook is authorized
	token := sess.GetLogin(c.Request)
	if token == nil || token.Label != hook.Repo.FullName {
		log.Errorf("invalid token sent with hook.")
		c.AbortWithStatus(403)
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

	switch {
	case repo.UserID == 0:
		log.Warnf("ignoring hook. repo %s has no owner.", repo.FullName)
		c.Writer.WriteHeader(204)
		return
	case !repo.PostCommit && hook.Commit.PullRequest != "":
		log.Infof("ignoring hook. repo %s is disabled.", repo.FullName)
		c.Writer.WriteHeader(204)
		return
	case !repo.PullRequest && hook.Commit.PullRequest == "":
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

	commit := hook.Commit
	commit.State = common.StatePending
	commit.RepoID = repo.ID

	// featch the .drone.yml file from the database
	raw, err := remote.Script(user, repo, commit)
	if err != nil {
		log.Errorf("failure to get .drone.yml for %s. %s", repo.FullName, err)
		c.Fail(404, err)
		return
	}
	// inject any private parameters into the .drone.yml
	if repo.Params != nil && len(repo.Params) != 0 {
		raw = []byte(inject.InjectSafe(string(raw), repo.Params))
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
		commit.Builds = append(commit.Builds, &common.Build{
			CommitID:    commit.ID,
			Sequence:    num + 1,
			State:       common.StatePending,
			Environment: axis,
		})
	}
	keys := &common.Keypair{
		Public:  repo.PublicKey,
		Private: repo.PrivateKey,
	}

	netrc, err := remote.Netrc(user)
	if err != nil {
		log.Errorf("failure to generate netrc for %s. %s", repo.FullName, err)
		c.Fail(500, err)
		return
	}

	// verify the branches can be built vs skipped
	when, _ := parser.ParseCondition(string(raw))
	if commit.PullRequest != "" && when != nil && !when.MatchBranch(commit.Branch) {
		log.Infof("ignoring hook. yaml file excludes repo and branch %s %s", repo.FullName, commit.Branch)
		c.AbortWithStatus(200)
		return
	}

	err = store.AddCommit(commit)
	if err != nil {
		log.Errorf("failure to save commit for %s. %s", repo.FullName, err)
		c.Fail(500, err)
		return
	}

	c.JSON(200, commit)

	err = remote.Status(user, repo, commit)
	if err != nil {
		log.Errorf("error setting commit status for %s/%d", repo.FullName, commit.Sequence)
	}

	queue_.Publish(&queue.Work{
		User:   user,
		Repo:   repo,
		Commit: commit,
		Keys:   keys,
		Netrc:  netrc,
		Yaml:   raw,
	})
}
