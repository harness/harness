package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/drone/drone/server/resource/build"
	"github.com/drone/drone/server/resource/commit"
	"github.com/drone/drone/server/resource/perm"
	"github.com/drone/drone/server/resource/repo"
	"github.com/drone/drone/server/session"
	"github.com/gorilla/pat"
)

type BuildHandler struct {
	builds  build.BuildManager
	commits commit.CommitManager
	repos   repo.RepoManager
	perms   perm.PermManager
	sess    session.Session
}

func NewBuildHandler(repos repo.RepoManager, commits commit.CommitManager, builds build.BuildManager,
	perms perm.PermManager, sess session.Session) *BuildHandler {
	return &BuildHandler{builds, commits, repos, perms, sess}
}

// GetCommit gets the commit for the repository, branch and sha.
// GET /v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}/builds/{build}
func (h *BuildHandler) GetBuild(w http.ResponseWriter, r *http.Request) error {
	var host, owner, name = parseRepo(r)
	var branch = r.FormValue(":branch")
	var sha = r.FormValue(":commit")
	var num, _ = strconv.Atoi(r.FormValue(":build"))

	// get the user form the session.
	user := h.sess.User(r)

	// get the repository from the database.
	repo, err := h.repos.FindName(host, owner, name)
	if err != nil {
		return notFound{err}
	}

	// user must have read access to the repository.
	if ok, _ := h.perms.Read(user, repo); !ok {
		return notFound{err}
	}

	// get the commit information for the specified hash
	commit, err := h.commits.FindSha(repo.ID, branch, sha)
	if err != nil {
		return notFound{err}
	}

	// get the builds for the hash
	build, err := h.builds.FindNumber(commit.ID, int64(num))
	if err != nil {
		return notFound{err}
	}

	return json.NewEncoder(w).Encode(build)
}

// GetBuilds gets the list of builds for a commit.
// GET /v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}/builds
func (h *BuildHandler) GetBuilds(w http.ResponseWriter, r *http.Request) error {
	var host, owner, name = parseRepo(r)
	var branch = r.FormValue(":branch")
	var sha = r.FormValue(":commit")

	// get the user form the session.
	user := h.sess.User(r)

	// get the repository from the database.
	repo, err := h.repos.FindName(host, owner, name)
	if err != nil {
		return notFound{err}
	}

	// user must have read access to the repository.
	if ok, _ := h.perms.Read(user, repo); !ok {
		return notFound{err}
	}

	// get the commit information for the specified hash
	commit, err := h.commits.FindSha(repo.ID, branch, sha)
	if err != nil {
		return notFound{err}
	}

	// get the builds for the hash
	builds, err := h.builds.List(commit.ID)
	if err != nil {
		return notFound{err}
	}

	return json.NewEncoder(w).Encode(builds)
}

// GetOut gets the console output for a build. If the build is in-progress it
// returns a link to the websocket (I think ...)
// GET /v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}/builds/{build}/out
func (h *BuildHandler) GetOut(w http.ResponseWriter, r *http.Request) error {
	var host, owner, name = parseRepo(r)
	var branch = r.FormValue(":branch")
	var sha = r.FormValue(":commit")
	var num, _ = strconv.Atoi(r.FormValue(":build"))

	// get the user form the session.
	user := h.sess.User(r)

	// get the repository from the database.
	repo, err := h.repos.FindName(host, owner, name)
	if err != nil {
		return notFound{err}
	}

	// user must have read access to the repository.
	if ok, _ := h.perms.Read(user, repo); !ok {
		return notFound{err}
	}

	// get the commit information for the specified hash
	commit, err := h.commits.FindSha(repo.ID, branch, sha)
	if err != nil {
		return notFound{err}
	}

	// get the builds for the hash
	out, err := h.builds.FindOutput(commit.ID, int64(num))
	if err != nil {
		return notFound{err}
	}

	w.Write(out)
	return nil
}

func (h *BuildHandler) Register(r *pat.Router) {
	r.Get("/v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}/builds/{build}/console", errorHandler(h.GetOut))
	r.Get("/v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}/builds/{build}", errorHandler(h.GetBuild))
	r.Get("/v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}/builds", errorHandler(h.GetBuilds))
}
