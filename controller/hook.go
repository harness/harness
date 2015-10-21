package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/engine"
	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/router/middleware/context"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/store"
	"github.com/drone/drone/yaml"
	"github.com/drone/drone/yaml/matrix"
)

func PostHook(c *gin.Context) {
	remote_ := remote.FromContext(c)

	tmprepo, build, err := remote_.Hook(c.Request)
	if err != nil {
		log.Errorf("failure to parse hook. %s", err)
		c.AbortWithError(400, err)
		return
	}
	if build == nil {
		c.Writer.WriteHeader(200)
		return
	}
	if tmprepo == nil {
		log.Errorf("failure to ascertain repo from hook.")
		c.Writer.WriteHeader(400)
		return
	}

	// a build may be skipped if the text [CI SKIP]
	// is found inside the commit message
	if strings.Contains(build.Message, "[CI SKIP]") {
		log.Infof("ignoring hook. [ci skip] found for %s")
		c.Writer.WriteHeader(204)
		return
	}

	repo, err := store.GetRepoOwnerName(c, tmprepo.Owner, tmprepo.Name)
	if err != nil {
		log.Errorf("failure to find repo %s/%s from hook. %s", tmprepo.Owner, tmprepo.Name, err)
		c.AbortWithError(404, err)
		return
	}

	// get the token and verify the hook is authorized
	parsed, err := token.ParseRequest(c.Request, func(t *token.Token) (string, error) {
		return repo.Hash, nil
	})
	if err != nil {
		log.Errorf("failure to parse token from hook for %s. %s", repo.FullName, err)
		c.AbortWithError(400, err)
		return
	}
	if parsed.Text != repo.FullName {
		log.Errorf("failure to verify token from hook. Expected %s, got %s", repo.FullName, parsed.Text)
		c.AbortWithStatus(403)
		return
	}

	if repo.UserID == 0 {
		log.Warnf("ignoring hook. repo %s has no owner.", repo.FullName)
		c.Writer.WriteHeader(204)
		return
	}
	var skipped = true
	if (build.Event == model.EventPush && repo.AllowPush) ||
		(build.Event == model.EventPull && repo.AllowPull) ||
		(build.Event == model.EventDeploy && repo.AllowDeploy) ||
		(build.Event == model.EventTag && repo.AllowTag) {
		skipped = false
	}

	if skipped {
		log.Infof("ignoring hook. repo %s is disabled for %s events.", repo.FullName, build.Event)
		c.Writer.WriteHeader(204)
		return
	}

	user, err := store.GetUser(c, repo.UserID)
	if err != nil {
		log.Errorf("failure to find repo owner %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}

	// if there is no email address associated with the pull request,
	// we lookup the email address based on the authors github login.
	//
	// my initial hesitation with this code is that it has the ability
	// to expose your email address. At the same time, your email address
	// is already exposed in the public .git log. So while some people will
	// a small number of people will probably be upset by this, I'm not sure
	// it is actually that big of a deal.
	if len(build.Email) == 0 {
		author, err := store.GetUserLogin(c, build.Author)
		if err == nil {
			build.Email = author.Email
		}
	}

	// if the remote has a refresh token, the current access token
	// may be stale. Therefore, we should refresh prior to dispatching
	// the job.
	if refresher, ok := remote_.(remote.Refresher); ok {
		ok, _ := refresher.Refresh(user)
		if ok {
			store.UpdateUser(c, user)
		}
	}

	// fetch the .drone.yml file from the database
	raw, sec, err := remote_.Script(user, repo, build)
	if err != nil {
		log.Errorf("failure to get .drone.yml for %s. %s", repo.FullName, err)
		c.AbortWithError(404, err)
		return
	}

	axes, err := matrix.Parse(string(raw))
	if err != nil {
		log.Errorf("failure to calculate matrix for %s. %s", repo.FullName, err)
		c.AbortWithError(400, err)
		return
	}
	if len(axes) == 0 {
		axes = append(axes, matrix.Axis{})
	}

	netrc, err := remote_.Netrc(user, repo)
	if err != nil {
		log.Errorf("failure to generate netrc for %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}

	key, _ := store.GetKey(c, repo)

	// verify the branches can be built vs skipped
	yconfig, _ := yaml.Parse(string(raw))
	var match = false
	for _, branch := range yconfig.Branches {
		if branch == build.Branch {
			match = true
			break
		}
		match, _ = filepath.Match(branch, build.Branch)
		if match {
			break
		}
	}
	if !match && len(yconfig.Branches) != 0 {
		log.Infof("ignoring hook. yaml file excludes repo and branch %s %s", repo.FullName, build.Branch)
		c.AbortWithStatus(200)
		return
	}

	// update some build fields
	build.Status = model.StatusPending
	build.RepoID = repo.ID

	// and use a transaction
	var jobs []*model.Job
	for num, axis := range axes {
		jobs = append(jobs, &model.Job{
			BuildID:     build.ID,
			Number:      num + 1,
			Status:      model.StatusPending,
			Environment: axis,
		})
	}
	err = store.CreateBuild(c, build, jobs...)
	if err != nil {
		log.Errorf("failure to save commit for %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, build)

	url := fmt.Sprintf("%s/%s/%d", httputil.GetURL(c.Request), repo.FullName, build.Number)
	err = remote_.Status(user, repo, build, url)
	if err != nil {
		log.Errorf("error setting commit status for %s/%d", repo.FullName, build.Number)
	}

	// get the previous build so taht we can send
	// on status change notifications
	last, _ := store.GetBuildLastBefore(c, repo, build.Branch, build.ID)

	engine_ := context.Engine(c)
	go engine_.Schedule(c.Copy(), &engine.Task{
		User:      user,
		Repo:      repo,
		Build:     build,
		BuildPrev: last,
		Jobs:      jobs,
		Keys:      key,
		Netrc:     netrc,
		Config:    string(raw),
		Secret:    string(sec),
		System: &model.System{
			Link:    httputil.GetURL(c.Request),
			Plugins: strings.Split(os.Getenv("PLUGIN_FILTER"), " "),
			Globals: strings.Split(os.Getenv("PLUGIN_PARAMS"), " "),
		},
	})

}
