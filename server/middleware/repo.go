package middleware

import (
	"net/http"
	"regexp"

	"github.com/drone/drone/server/datastore"
	"github.com/goji/context"
	"github.com/zenazn/goji/web"
)

// SetRepo is a middleware function that retrieves
// the repository and stores in the context.
func SetRepo(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var (
			ctx   = context.FromC(*c)
			host  = c.URLParams["host"]
			owner = c.URLParams["owner"]
			name  = c.URLParams["name"]
			user  = ToUser(c)
		)

		repo, err := datastore.GetRepoName(ctx, host, owner, name)
		switch {
		case err != nil && user == nil:
			w.WriteHeader(http.StatusUnauthorized)
			return
		case err != nil && user != nil:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		role, _ := datastore.GetPerm(ctx, user, repo)
		RepoToC(c, repo)
		RoleToC(c, role)
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// RequireRepoRead is a middleware function that verifies
// the user has read access to the repository.
func RequireRepoRead(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var (
			role = ToRole(c)
			user = ToUser(c)
		)
		switch {
		case role == nil:
			w.WriteHeader(http.StatusInternalServerError)
		case user == nil && role.Read == false:
			w.WriteHeader(http.StatusUnauthorized)
			return
		case user == nil && role.Read == false:
			w.WriteHeader(http.StatusUnauthorized)
			return
		case user != nil && role.Read == false:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// RequireRepoAdmin is a middleware function that verifies
// the user has admin access to the repository.
func RequireRepoAdmin(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var (
			role = ToRole(c)
			user = ToUser(c)
		)

		// Admin access is only rquired for POST, PUT, DELETE methods.
		// If this is a GET request we can proceed immediately.
		if r.Method == "GET" {
			h.ServeHTTP(w, r)
			return
		}

		switch {
		case role == nil:
			w.WriteHeader(http.StatusInternalServerError)
			return
		case user == nil && role.Admin == false:
			w.WriteHeader(http.StatusUnauthorized)
			return
		case user != nil && role.Read == false && role.Admin == false:
			w.WriteHeader(http.StatusNotFound)
			return
		case user != nil && role.Write == true && role.Admin == false:
			if IsRebuild(r.URL.Path) {
				h.ServeHTTP(w, r)
				return
			}
			w.WriteHeader(http.StatusForbidden)
			return
		case user != nil && role.Read == true && role.Admin == false:
			w.WriteHeader(http.StatusForbidden)
			return
		default:
			h.ServeHTTP(w, r)
			return
		}

	}
	return http.HandlerFunc(fn)
}

func IsRebuild(path string) bool {
	const pattern = `\/(.*)\/(.*)\/(.*)\/branches\/(.*)\/commits\/(.*)`
	ok, _ := regexp.MatchString(pattern, path)
	return ok
}
