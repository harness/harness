package handler

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/server/worker"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/model"
	"github.com/goji/context"
	"github.com/zenazn/goji/web"
)

// GetCommitList accepts a request to retrieve a list
// of recent commits by Repo, and retur in JSON format.
//
//     GET /api/repos/:host/:owner/:name/commits?limit=:limit&offset=:offset
//
func GetCommitList(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var repo = ToRepo(c)
	var limit = ToLimit(r)
	var offset = ToOffset(r)

	commits, err := datastore.GetCommitList(ctx, repo, limit, offset)
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

// PostHook accepts a post-commit hook and parses the payload
// in order to trigger a build. The payload is specified to the
// remote system (ie GitHub) and will therefore get parsed by
// the appropriate remote plugin.
//
//     POST /api/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}
//
func PostCommit(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var (
		branch = c.URLParams["branch"]
		hash   = c.URLParams["commit"]
		host   = c.URLParams["host"]
		repo   = ToRepo(c)
		remote = remote.Lookup(host)
	)

	commit, err := datastore.GetCommitSha(ctx, repo, branch, hash)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if commit.Status == model.StatusStarted ||
		commit.Status == model.StatusEnqueue {
		w.WriteHeader(http.StatusConflict)
		return
	}

	commit.Status = model.StatusEnqueue
	commit.Started = 0
	commit.Finished = 0
	commit.Duration = 0
	if err := datastore.PutCommit(ctx, commit); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	owner, err := datastore.GetUser(ctx, repo.UserID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Request a new token and update
	user_token, err := remote.GetToken(owner)
	if user_token != nil {
		owner.Access = user_token.AccessToken
		owner.Secret = user_token.RefreshToken
		owner.TokenExpiry = user_token.Expiry
		datastore.PutUser(ctx, owner)
	} else if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// drop the items on the queue
	go worker.Do(ctx, &worker.Work{
		User:   owner,
		Repo:   repo,
		Commit: commit,
		Host:   httputil.GetURL(r),
	})

	w.WriteHeader(http.StatusOK)
}
