package handler

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/shared/model"
	"github.com/goji/context"
	"github.com/zenazn/goji/web"
)

// GetUsers accepts a request to retrieve all users
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

	account := model.NewUser(host, login, "")
	if err := datastore.PostUser(ctx, account); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(account)
}

// DeleteUser accepts a request to delete the specified
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
