package handler

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/server/resource/commit"
	"github.com/drone/drone/server/resource/perm"
	"github.com/drone/drone/server/resource/repo"
	"github.com/drone/drone/server/session"
	"github.com/gorilla/pat"
)

type BranchHandler struct {
	perms   perm.PermManager
	repos   repo.RepoManager
	commits commit.CommitManager
	sess    session.Session
}

func NewBranchHandler(repos repo.RepoManager, commits commit.CommitManager, perms perm.PermManager, sess session.Session) *BranchHandler {
	return &BranchHandler{perms, repos, commits, sess}
}

// GetBranches gets a list of all branches and their most recent commits.
// GET /v1/repos/{host}/{owner}/{name}/branches
func (h *BranchHandler) GetBranches(w http.ResponseWriter, r *http.Request) error {
	var host, owner, name = parseRepo(r)

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

	branches, err := h.commits.ListBranches(repo.ID)
	if err != nil {
		return notFound{err}
	}

	return json.NewEncoder(w).Encode(branches)
}

func (h *BranchHandler) Register(r *pat.Router) {
	r.Get("/v1/repos/{host}/{owner}/{name}/branches", errorHandler(h.GetBranches))
}
