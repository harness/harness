package handler

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/server/datastore"
	"github.com/goji/context"
	"github.com/zenazn/goji/web"
)

// GetCommitList accepts a request to retrieve a list
// of recent commits by Repo, and retur in JSON format.
//
//     GET /api/repos/:host/:owner/:name/commits
//
func GetCommitList(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var repo = ToRepo(c)

	commits, err := datastore.GetCommitList(ctx, repo)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(commits)
}

// GetCommit accepts a request to retrieve a commit
// from the datastore for the given repository, branch and
// commit hash.
//
//     GET /api/repos/:host/:owner/:name/branches/:branch/commits/:commit
//
func GetCommit(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var (
		branch = c.URLParams["branch"]
		hash   = c.URLParams["commit"]
		repo   = ToRepo(c)
	)

	commit, err := datastore.GetCommitSha(ctx, repo, branch, hash)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(commit)
}

func PostCommit(c web.C, w http.ResponseWriter, r *http.Request) {}

/*
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

	repoOwner, err := h.users.Find(repo.UserID)
	if err != nil {
		return badRequest{err}
	}

	// drop the items on the queue
	// drop the items on the queue
	go func() {
		h.queue <- &model.Request{
			User:   repoOwner,
			Host:   httputil.GetURL(r),
			Repo:   repo,
			Commit: c,
		}
	}()

	w.WriteHeader(http.StatusOK)
	return nil
}
*/
