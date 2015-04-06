package handler

import (
	"net/http"
	"strconv"

	"github.com/drone/drone/shared/model"
	"github.com/zenazn/goji/web"
)

// ToUser returns the User from the current
// request context. If the User does not exist
// a nil value is returned.
func ToUser(c web.C) *model.User {
	var v = c.Env["user"]
	if v == nil {
		return nil
	}
	u, ok := v.(*model.User)
	if !ok {
		return nil
	}
	return u
}

// ToRepo returns the Repo from the current
// request context. If the Repo does not exist
// a nil value is returned.
func ToRepo(c web.C) *model.Repo {
	var v = c.Env["repo"]
	if v == nil {
		return nil
	}
	r, ok := v.(*model.Repo)
	if !ok {
		return nil
	}
	return r
}

// ToRole returns the Role from the current
// request context. If the Role does not exist
// a nil value is returned.
func ToRole(c web.C) *model.Perm {
	var v = c.Env["role"]
	if v == nil {
		return nil
	}
	p, ok := v.(*model.Perm)
	if !ok {
		return nil
	}
	return p
}

// ToLimit returns the Limit from current request
// query if limit doesn't present set default offset
// equal to 20, maximum limit equal 100
func ToLimit(r *http.Request) int {
	if len(r.FormValue("limit")) == 0 {
		return 20
	}

	limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil {
		return 20
	}

	if limit > 100 {
		return 100
	}

	return limit
}

// ToOffset returns the Offset from current request
// query if offset doesn't present set default offset
// equal to 0
func ToOffset(r *http.Request) int {
	if len(r.FormValue("offset")) == 0 {
		return 0
	}

	offset, err := strconv.Atoi(r.FormValue("offset"))
	if err != nil {
		return 0
	}

	return offset
}
