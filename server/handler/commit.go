package handler

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/model"
	"github.com/gorilla/pat"
)

type CommitHandler struct {
	perms   database.PermManager
	repos   database.RepoManager
	commits database.CommitManager
	sess    session.Session
	queue   chan *model.Request
}

func NewCommitHandler(repos database.RepoManager, commits database.CommitManager, perms database.PermManager, sess session.Session, queue chan *model.Request) *CommitHandler {
	return &CommitHandler{perms, repos, commits, sess, queue}
}

// GetFeed gets recent commits for the repository and branch
// GET /v1/repos/{host}/{owner}/{name}/branches/{branch}/commits
func (h *CommitHandler) GetFeed(w http.ResponseWriter, r *http.Request) error {
	var host, owner, name = parseRepo(r)
	var branch = r.FormValue(":branch")

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

	commits, err := h.commits.ListBranch(repo.ID, branch)
	if err != nil {
		return notFound{err}
	}

	return json.NewEncoder(w).Encode(commits)
}

// GetCommit gets the commit for the repository, branch and sha.
// GET /v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}
func (h *CommitHandler) GetCommit(w http.ResponseWriter, r *http.Request) error {
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

	commit, err := h.commits.FindSha(repo.ID, branch, sha)
	if err != nil {
		return notFound{err}
	}

	return json.NewEncoder(w).Encode(commit)
}

// GetCommitOutput gets the commit's stdout.
// GET /v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}/console
func (h *CommitHandler) GetCommitOutput(w http.ResponseWriter, r *http.Request) error {
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

	commit, err := h.commits.FindSha(repo.ID, branch, sha)
	if err != nil {
		return notFound{err}
	}

	output, err := h.commits.FindOutput(commit.ID)
	if err != nil {
		return notFound{err}
	}

	w.Write(output)
	return nil
}

// PostCommit gets the commit for the repository and schedules to re-build.
// GET /v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}
func (h *CommitHandler) PostCommit(w http.ResponseWriter, r *http.Request) error {
	var host, owner, name = parseRepo(r)
	var branch = r.FormValue(":branch")
	var sha = r.FormValue(":commit")

	// get the user form the session.
	user := h.sess.User(r)
	if user == nil {
		return notAuthorized{}
	}

	// get the repo from the database
	repo, err := h.repos.FindName(host, owner, name)
	if err != nil {
		return notFound{err}
	}

	// user must have admin access to the repository.
	if ok, _ := h.perms.Admin(user, repo); !ok {
		return notFound{err}
	}

	c, err := h.commits.FindSha(repo.ID, branch, sha)
	if err != nil {
		return notFound{err}
	}

	// we can't start an already started build
	if c.Status == model.StatusStarted || c.Status == model.StatusEnqueue {
		return badRequest{}
	}

	c.Status = model.StatusEnqueue
	c.Started = 0
	c.Finished = 0
	c.Duration = 0
	if err := h.commits.Update(c); err != nil {
		return internalServerError{err}
	}

	owner, err := h.users.Find(repo.UserID)
	if err != nil {
		return badRequest{err}
	}

	// drop the items on the queue
	// drop the items on the queue
	go func() {
		h.queue <- &model.Request{
			User:   owner,
			Host:   httputil.GetURL(r),
			Repo:   repo,
			Commit: c,
		}
	}()

	w.WriteHeader(http.StatusOK)
	return nil
}

func (h *CommitHandler) Register(r *pat.Router) {
	r.Get("/v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}/console", errorHandler(h.GetCommitOutput))
	r.Get("/v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}", errorHandler(h.GetCommit))
	r.Post("/v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}", errorHandler(h.PostCommit)).Queries("action", "rebuild")
	r.Get("/v1/repos/{host}/{owner}/{name}/branches/{branch}/commits", errorHandler(h.GetFeed))
}
