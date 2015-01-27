package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/server/sync"
	"github.com/drone/drone/shared/model"
	"github.com/goji/context"
	"github.com/zenazn/goji/web"
)

// GetLogin accepts a request to authorize the user and to
// return a valid OAuth2 access token. The access token is
// returned as url segment #access_token
//
//     GET /login/:host
//
func GetLogin(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var host = c.URLParams["host"]
	var redirect = "/"
	var remote = remote.Lookup(host)
	if remote == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Del("Content-Type")

	// authenticate the user
	login, err := remote.Authorize(w, r)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if login == nil {
		// in this case we probably just redirected
		// the user, so we can exit with no error
		return
	}

	// get the user from the database
	u, err := datastore.GetUserLogin(ctx, host, login.Login)
	if err != nil {
		// if self-registration is disabled we should
		// return a notAuthorized error. the only exception
		// is if no users exist yet in the system we'll proceed.
		if remote.OpenRegistration() == false {
			users, err := datastore.GetUserList(ctx)
			if err != nil || len(users) != 0 {
				log.Println("Unable to create account. Registration is closed")
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}

		// create the user account
		u = model.NewUser(remote.GetKind(), login.Login, login.Email)
		u.Name = login.Name
		u.SetEmail(login.Email)

		// insert the user into the database
		if err := datastore.PostUser(ctx, u); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// the user id should NEVER equal zero
		if u.ID == 0 {
			log.Println("Unable to create account. User ID is zero")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// if this is the first user, they
		// should be an admin.
		if u.ID == 1 {
			u.Admin = true
		}
	}

	// update the user access token
	// in case it changed in GitHub
	u.Access = login.Access
	u.Secret = login.Secret
	u.Name = login.Name
	u.TokenExpiry = login.Expiry
	u.SetEmail(login.Email)
	u.Syncing = u.IsStale()

	if err := datastore.PutUser(ctx, u); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// look at the last synchronized date to determine if
	// we need to re-sync the account.
	//
	// todo(bradrydzewski) this should move to a server/sync package and
	//      should be injected into this struct, just like the database code.
	//
	// todo(bradrydzewski) this login should be a bit more intelligent
	//      than the current implementation.
	if u.Syncing {
		redirect = "/sync"
		log.Println("sync user account.", u.Login)

		// sync inside a goroutine
		go sync.SyncUser(ctx, u, remote)
	}

	token, err := session.GenerateToken(ctx, r, u)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	redirect = redirect + "#access_token=" + token

	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

// GetLoginList accepts a request to retrive a list of
// all OAuth login options.
//
//     GET /api/remotes/login
//
func GetLoginList(c web.C, w http.ResponseWriter, r *http.Request) {
	var list = remote.Registered()
	var logins []interface{}
	for _, item := range list {
		logins = append(logins, struct {
			Type string `json:"type"`
			Host string `json:"host"`
		}{item.GetKind(), item.GetHost()})
	}
	json.NewEncoder(w).Encode(&logins)
}
