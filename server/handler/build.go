package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/model"
	"github.com/gorilla/pat"
)

type BuildHandler struct {
	users   database.UserManager
	perms   database.PermManager
	repos   database.RepoManager
	commits database.CommitManager
	builds  database.BuildManager
	sess    session.Session
	queue   chan *model.Request
}

func NewBuildHandler(users database.UserManager, repos database.RepoManager, commits database.CommitManager, builds database.BuildManager, perms database.PermManager, sess session.Session, queue chan *model.Request) *BuildHandler {
	return &BuildHandler{users, perms, repos, commits, builds, sess, queue}
}

func (h *BuildHandler) GetFeed(w http.ResponseWriter, r *http.Request) error {
	var host, owner, name = parseRepo(r)
	var branch = r.FormValue(":branch")
	var sha = r.FormValue(":commit")

	// get the user form the session.
	user := h.sess.User(r)

	// get the repository from the database.
	repo, err := h.repos.FindName(host, owner, name)
	switch {
	case err != nil && user == nil:
		return notAuthorized{}
	case err != nil && user != nil:
		return notFound{}
	}

	// user must have read access to the repository.
	ok, _ := h.perms.Read(user, repo)
	switch {
	case ok == false && user == nil:
		return notAuthorized{}
	case ok == false && user != nil:
		return notFound{}
	}

	c, err := h.commits.FindSha(repo.ID, branch, sha)
	if err != nil {
		return notFound{err}
	}

	builds, err := h.builds.FindCommit(c.ID)
	if err != nil {
		return notFound{err}
	}

	return json.NewEncoder(w).Encode(builds)
}

func (h *BuildHandler) GetBuild(w http.ResponseWriter, r *http.Request) error {
	var host, owner, name = parseRepo(r)
	var branch = r.FormValue(":branch")
	var sha = r.FormValue(":commit")

	build_id, err := strconv.Atoi(r.FormValue(":build"))
	if err != nil {
		return notFound{}
	}

	// get the user form the session.
	user := h.sess.User(r)

	// get the repository from the database.
	repo, err := h.repos.FindName(host, owner, name)
	switch {
	case err != nil && user == nil:
		return notAuthorized{}
	case err != nil && user != nil:
		return notFound{}
	}

	// user must have read access to the repository.
	ok, _ := h.perms.Read(user, repo)
	switch {
	case ok == false && user == nil:
		return notAuthorized{}
	case ok == false && user != nil:
		return notFound{}
	}

	c, err := h.commits.FindSha(repo.ID, branch, sha)
	if err != nil || c == nil {
		return notFound{err}
	}

	build, err := h.builds.Find(int64(build_id), c.ID)
	if err != nil {
		log.Println(err)
		return notFound{err}
	}

	return json.NewEncoder(w).Encode(build)
}

func (h *BuildHandler) PostBuild(w http.ResponseWriter, r *http.Request) error {
	var host, owner, name = parseRepo(r)
	var branch = r.FormValue(":branch")
	var sha = r.FormValue(":commit")

	build_id, err := strconv.Atoi(r.FormValue(":build"))
	if err != nil {
		return notFound{}
	}

	// get the user form the session.
	user := h.sess.User(r)
	if user == nil {
		return notAuthorized{}
	}

	// get the repo from the database
	repo, err := h.repos.FindName(host, owner, name)
	switch {
	case err != nil && user == nil:
		return notAuthorized{}
	case err != nil && user != nil:
		return notFound{}
	}

	// user must have admin access to the repository.
	if ok, _ := h.perms.Admin(user, repo); !ok {
		return notFound{err}
	}

	c, err := h.commits.FindSha(repo.ID, branch, sha)
	if err != nil {
		return notFound{err}
	}

	b, err := h.builds.Find(int64(build_id), c.ID)
	if err != nil {
		return notFound{err}
	}

	// we can't start an already started build
	if c.Status == model.StatusStarted || c.Status == model.StatusEnqueue {
		return badRequest{errors.New("This commit already builds")}
	}

	if b.Status == model.StatusStarted || b.Status == model.StatusEnqueue {
		return badRequest{errors.New("This build already builds")}
	}

	c.Status = model.StatusEnqueue
	c.Started = 0
	c.Finished = 0
	c.Duration = 0
	if err := h.commits.Update(c); err != nil {
		return internalServerError{err}
	}

	b.Status = model.StatusEnqueue
	b.Started = 0
	b.Finished = 0
	b.Duration = 0
	if err := h.builds.Update(b); err != nil {
		return internalServerError{err}
	}

	repoOwner, err := h.users.Find(repo.UserID)
	if err != nil {
		return badRequest{err}
	}

	var builds []*model.Build
	builds = append(builds, b)

	// drop the items on the queue
	// drop the items on the queue
	go func() {
		h.queue <- &model.Request{
			User:   repoOwner,
			Host:   httputil.GetURL(r),
			Repo:   repo,
			Commit: c,
			Builds: builds,
		}
	}()

	w.WriteHeader(http.StatusOK)
	return nil
}

func (h *BuildHandler) Register(r *pat.Router) {
	r.Get("/v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}/builds/{build}", errorHandler(h.GetBuild))
	r.Post("/v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}/builds/{build}", errorHandler(h.PostBuild)).Queries("action", "rebuild")
	r.Get("/v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}/builds", errorHandler(h.GetFeed))
}
