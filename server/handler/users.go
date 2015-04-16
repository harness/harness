package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/server/sync"
	"github.com/drone/drone/shared/model"
	"github.com/goji/context"
	"github.com/zenazn/goji/web"
)

// GetUserList accepts a request to retrieve all users
// from the datastore and return encoded in JSON format.
//
//     GET /api/users
//
func GetUserList(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)

	users, err := datastore.GetUserList(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(users)
}

// GetUser accepts a request to retrieve a user by hostname
// and login from the datastore and return encoded in JSON
// format.
//
//     GET /api/users/:host/:login
//
func GetUser(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var (
		user  = ToUser(c)
		host  = c.URLParams["host"]
		login = c.URLParams["login"]
	)

	user, err := datastore.GetUserLogin(ctx, host, login)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(user)
}

// PostUser accepts a request to create a new user in the
// system. The created user account is returned in JSON
// format if successful.
//
//     POST /api/users/:host/:login
//
func PostUser(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var (
		host  = c.URLParams["host"]
		login = c.URLParams["login"]
	)
	var remote = remote.Lookup(host)
	if remote == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// not sure I love this, but POST now flexibly accepts the oauth_token for
	// GitHub as either application/x-www-form-urlencoded OR as applcation/json
	// with this format:
	//    { "oauth_token": "...." }
	var oauthToken string
	switch cnttype := r.Header.Get("Content-Type"); cnttype {
	case "application/json":
		var out interface{}
		err := json.NewDecoder(r.Body).Decode(&out)
		if err == nil {
			if val, ok := out.(map[string]interface{})["oauth_token"]; ok {
				oauthToken = val.(string)
			}
		}
	case "application/x-www-form-urlencoded":
		oauthToken = r.PostForm.Get("oauth_token")
	default:
		// we don't recognize the content-type, but it isn't worth it
		// to error here
		log.Printf("PostUser(%s) Unknown 'Content-Type': %s)", r.URL, cnttype)
	}
	account := model.NewUser(host, login, "", oauthToken)

	if err := datastore.PostUser(ctx, account); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// borrowed this concept from login.go. upon first creation we
	// may trying syncing the user's repositories.
	account.Syncing = account.IsStale()
	if err := datastore.PutUser(ctx, account); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if account.Syncing {
		log.Println("sync user account.", account.Login)

		// sync inside a goroutine
		go sync.SyncUser(ctx, account, remote)
	}

	json.NewEncoder(w).Encode(account)
}

// DelUser accepts a request to delete the specified
// user account from the system. A successful request will
// respond with an OK 200 status.
//
//     DELETE /api/users/:host/:login
//
func DelUser(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var (
		user  = ToUser(c)
		host  = c.URLParams["host"]
		login = c.URLParams["login"]
	)

	account, err := datastore.GetUserLogin(ctx, host, login)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if account.ID == user.ID {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := datastore.DelUser(ctx, account); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
