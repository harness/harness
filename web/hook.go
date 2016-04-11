package web

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/engine"
	"github.com/drone/drone/engine/parser"
	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/router/middleware/context"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/store"
)

var (
	droneYml = os.Getenv("BUILD_CONFIG_FILE")
	droneSec string
)

func init() {
	if droneYml == "" {
		droneYml = ".drone.yml"
	}
	droneSec = fmt.Sprintf("%s.sec", strings.TrimSuffix(droneYml, filepath.Ext(droneYml)))
}

var skipRe = regexp.MustCompile(`\[(?i:ci *skip|skip *ci)\]`)

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

	// skip the build if any case-insensitive combination of the words "skip" and "ci"
	// wrapped in square brackets appear in the commit message
	skipMatch := skipRe.FindString(build.Message)
	if len(skipMatch) > 0 {
		log.Infof("ignoring hook. %s found in %s", skipMatch, build.Commit)
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

	// fetch the build file from the database
	raw, err := remote_.File(user, repo, build, droneYml)
	if err != nil {
		log.Errorf("failure to get build config for %s. %s", repo.FullName, err)
		c.AbortWithError(404, err)
		return
	}
	sec, err := remote_.File(user, repo, build, droneSec)
	if err != nil {
		log.Debugf("cannot find build secrets for %s. %s", repo.FullName, err)
		// NOTE we don't exit on failure. The sec file is optional
	}

	axes, err := parser.ParseMatrix(raw)
	if err != nil {
		c.String(500, "Failed to parse yaml file or calculate matrix. %s", err)
		return
	}
	if len(axes) == 0 {
		axes = append(axes, parser.Axis{})
	}

	netrc, err := remote_.Netrc(user, repo)
	if err != nil {
		c.String(500, "Failed to generate netrc file. %s", err)
		return
	}

	key, _ := store.GetKey(c, repo)

	// verify the branches can be built vs skipped
	branches := parser.ParseBranch(raw)
	if !branches.Matches(build.Branch) {
		c.String(200, "Branch does not match restrictions defined in yaml")
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

	// get the previous build so that we can send
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
			Link:      httputil.GetURL(c.Request),
			Plugins:   strings.Split(os.Getenv("PLUGIN_FILTER"), " "),
			Globals:   strings.Split(os.Getenv("PLUGIN_PARAMS"), " "),
			Escalates: strings.Split(os.Getenv("ESCALATE_FILTER"), " "),
		},
	})

}
