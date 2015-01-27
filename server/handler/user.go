package handler

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/server/sync"
	"github.com/drone/drone/shared/model"
	"github.com/goji/context"
	"github.com/zenazn/goji/web"
)

// GetUserCurrent accepts a request to retrieve the
// currently authenticated user from the datastore
// and return in JSON format.
//
//     GET /api/user
//
func GetUserCurrent(c web.C, w http.ResponseWriter, r *http.Request) {
	var user = ToUser(c)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// return private data for the currently authenticated
	// user, specifically, their auth token.
	data := struct {
		*model.User
		Token string `json:"token"`
	}{user, user.Token}
	json.NewEncoder(w).Encode(&data)
}

// PutUser accepts a request to update the currently
// authenticated User profile.
//
//     PUT /api/user
//
func PutUser(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var user = ToUser(c)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// unmarshal the repository from the payload
	defer r.Body.Close()
	in := model.User{}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// update the user email
	if len(in.Email) != 0 {
		user.SetEmail(in.Email)
	}
	// update the user full name
	if len(in.Name) != 0 {
		user.Name = in.Name
	}

	// update the database
	if err := datastore.PutUser(ctx, user); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(user)
}

// GetRepos accepts a request to get the currently
// authenticated user's repository list from the datastore,
// encoded and returned in JSON format.
//
//     GET /api/user/repos
//
func GetUserRepos(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var user = ToUser(c)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	repos, err := datastore.GetRepoList(ctx, user)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(&repos)
}

// GetUserFeed accepts a request to get the user's latest
// build feed, across all repositories, from the datastore.
// The results are encoded and returned in JSON format.
//
//     GET /api/user/feed
//
func GetUserFeed(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var user = ToUser(c)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	repos, err := datastore.GetCommitListUser(ctx, user)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(&repos)
}

// GetUserActivity accepts a request to get the user's latest
// build activity, across all repositories, from the datastore.
// The results are encoded and returned in JSON format.
//
//     GET /api/user/activity
//
func GetUserActivity(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var user = ToUser(c)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	repos, err := datastore.GetCommitListActivity(ctx, user)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(&repos)
}

// PostUserSync accepts a request to post user sync
//
//     POST /api/user/sync
//
func PostUserSync(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var user = ToUser(c)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var remote = remote.Lookup(user.Remote)
	if remote == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if user.Syncing {
		w.WriteHeader(http.StatusConflict)
		return
	}

	// Request a new token and update
	user_token, err := remote.GetToken(user)
	if user_token != nil {
		user.Access = user_token.AccessToken
		user.Secret = user_token.RefreshToken
		user.TokenExpiry = user_token.Expiry
	} else if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	user.Syncing = true
	if err := datastore.PutUser(ctx, user); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	go sync.SyncUser(ctx, user, remote)
	w.WriteHeader(http.StatusNoContent)
	return
}
