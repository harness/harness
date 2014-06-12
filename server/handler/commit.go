package handler

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/server/queue"
	"github.com/drone/drone/server/resource/commit"
	"github.com/drone/drone/server/resource/perm"
	"github.com/drone/drone/server/resource/repo"
	"github.com/drone/drone/server/session"
	"github.com/gorilla/pat"
)

type CommitHandler struct {
	perms   perm.PermManager
	repos   repo.RepoManager
	commits commit.CommitManager
	sess    session.Session
	queue   *queue.Queue
}

func NewCommitHandler(repos repo.RepoManager, commits commit.CommitManager, perms perm.PermManager, sess session.Session, queue *queue.Queue) *CommitHandler {
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
	if c.Status == commit.StatusStarted || c.Status == commit.StatusEnqueue {
		return badRequest{}
	}

	c.Status = commit.StatusEnqueue
	c.Started = 0
	c.Finished = 0
	c.Duration = 0
	if err := h.commits.Update(c); err != nil {
		return internalServerError{err}
	}

	// drop the items on the queue
	h.queue.Add(&queue.BuildTask{Repo: repo, Commit: c})
	return nil

	return notImplemented{}
}

func (h *CommitHandler) Register(r *pat.Router) {
	r.Get("/v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}/console", errorHandler(h.GetCommitOutput))
	r.Get("/v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}", errorHandler(h.GetCommit))
	r.Post("/v1/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}", errorHandler(h.PostCommit)).Queries("action", "rebuild")
	r.Get("/v1/repos/{host}/{owner}/{name}/branches/{branch}/commits", errorHandler(h.GetFeed))
}
