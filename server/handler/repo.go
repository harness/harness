package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/model"
	"github.com/drone/drone/shared/sshutil"
	"github.com/goji/context"
	"github.com/zenazn/goji/web"
)

// GetRepo accepts a request to retrieve a commit
// from the datastore for the given repository, branch and
// commit hash.
//
//     GET /api/repos/:host/:owner/:name
//
func GetRepo(c web.C, w http.ResponseWriter, r *http.Request) {
	var (
		role = ToRole(c)
		repo = ToRepo(c)
	)

	// if the user is not requesting (or cannot access)
	// admin data then we just return the repo as-is
	if role.Admin == false {
		json.NewEncoder(w).Encode(repo)
		return
	}

	// else we should return restricted fields
	json.NewEncoder(w).Encode(struct {
		*model.Repo
		PublicKey string      `json:"public_key"`
		Params    string      `json:"params"`
		Perm      *model.Perm `json:"role"`
	}{repo, repo.PublicKey, repo.Params, role})
}

// DelRepo accepts a request to inactivate the named
// repository. This will disable all builds in the system
// for this repository.
//
//     DEL /api/repos/:host/:owner/:name
//
func DelRepo(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var repo = ToRepo(c)

	// disable everything
	repo.Active = false
	repo.PullRequest = false
	repo.PostCommit = false
	repo.UserID = 0

	if err := datastore.PutRepo(ctx, repo); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PostRepo accapets a request to activate the named repository
// in the datastore. It returns a 201 status created if successful
//
//     POST /api/repos/:host/:owner/:name
//
func PostRepo(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var repo = ToRepo(c)
	var user = ToUser(c)

	// update the repo active flag and fields
	repo.Active = true
	repo.PullRequest = true
	repo.PostCommit = true
	repo.UserID = user.ID
	repo.Timeout = 3600 // default to 1 hour

	// generate a secret key for post-commit hooks
	if len(repo.Token) == 0 {
		repo.Token = model.GenerateToken()
	}

	// generates the rsa key
	if len(repo.PublicKey) == 0 || len(repo.PrivateKey) == 0 {
		key, err := sshutil.GeneratePrivateKey()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		repo.PublicKey = sshutil.MarshalPublicKey(&key.PublicKey)
		repo.PrivateKey = sshutil.MarshalPrivateKey(key)
	}

	var remote = remote.Lookup(repo.Host)
	if remote == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Request a new token and update
	user_token, err := remote.GetToken(user)
	if user_token != nil {
		user.Access = user_token.AccessToken
		user.Secret = user_token.RefreshToken
		user.TokenExpiry = user_token.Expiry
		datastore.PutUser(ctx, user)
	} else if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// setup the post-commit hook with the remote system and
	// if necessary, register the public key
	var hook = fmt.Sprintf("%s/api/hook/%s/%s", httputil.GetURL(r), repo.Remote, repo.Token)
	if err := remote.Activate(user, repo, hook); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := datastore.PutRepo(ctx, repo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(repo)
}

// PutRepo accapets a request to update the named repository
// in the datastore. It expects a JSON input and returns the
// updated repository in JSON format if successful.
//
//     PUT /api/repos/:host/:owner/:name
//
func PutRepo(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var repo = ToRepo(c)
	var user = ToUser(c)

	// unmarshal the repository from the payload
	defer r.Body.Close()
	in := struct {
		PostCommit  *bool   `json:"post_commits"`
		PullRequest *bool   `json:"pull_requests"`
		Privileged  *bool   `json:"privileged"`
		Params      *string `json:"params"`
		Timeout     *int64  `json:"timeout"`
		PublicKey   *string `json:"public_key"`
		PrivateKey  *string `json:"private_key"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if in.Params != nil {
		repo.Params = *in.Params
		if _, err := repo.ParamMap(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if in.PostCommit != nil {
		repo.PostCommit = *in.PostCommit
	}
	if in.PullRequest != nil {
		repo.PullRequest = *in.PullRequest
	}
	if in.Privileged != nil && user.Admin {
		repo.Privileged = *in.Privileged
	}
	if in.Timeout != nil && user.Admin {
		repo.Timeout = *in.Timeout
	}
	if in.PrivateKey != nil && in.PublicKey != nil {
		repo.PublicKey = *in.PublicKey
		repo.PrivateKey = *in.PrivateKey
	}
	if err := datastore.PutRepo(ctx, repo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(repo)
}
