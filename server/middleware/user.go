package middleware

import (
	"net/http"

	"github.com/drone/drone/server/session"
	"github.com/goji/context"
	"github.com/zenazn/goji/web"
)

// SetUser is a middleware function that retrieves
// the currently authenticated user from the request
// and stores in the context.
func SetUser(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var ctx = context.FromC(*c)
		var user = session.GetUser(ctx, r)
		if user != nil && user.ID != 0 {
			UserToC(c, user)
		}
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// RequireUser is a middleware function that verifies
// there is a currently authenticated user stored in
// the context.
func RequireUser(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if ToUser(c) == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// RequireUserAdmin is a middleware function that verifies
// there is a currently authenticated user stored in
// the context with ADMIN privilege.
func RequireUserAdmin(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var user = ToUser(c)
		switch {
		case user == nil:
			w.WriteHeader(http.StatusUnauthorized)
			return
		case user != nil && !user.Admin:
			w.WriteHeader(http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
