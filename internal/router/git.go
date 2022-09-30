package router

import (
	"fmt"
	"net/http"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/middleware/accesslog"
	middleware_authn "github.com/harness/gitness/internal/api/middleware/authn"
	"github.com/harness/gitness/internal/api/middleware/resolve"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/enum"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog/hlog"
)

/*
 * newGitHandler returns a new http handler for handling GIT calls.
 */
func newGitHandler(
	_ store.SystemStore,
	_ store.UserStore,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	authenticator authn.Authenticator,
	authorizer authz.Authorizer) http.Handler {
	guard := guard.New(authorizer, spaceStore, repoStore)

	// Use go-chi router for inner routing (restricted to mountPath!)
	r := chi.NewRouter()
	// Apply common api middleware
	r.Use(middleware.NoCache)
	r.Use(middleware.Recoverer)

	// configure logging middleware.
	r.Use(hlog.URLHandler("url"))
	r.Use(hlog.MethodHandler("method"))
	r.Use(hlog.RequestIDHandler("request", "Request-Id"))
	r.Use(accesslog.HlogHandler())

	// for now always attempt auth - enforced per operation
	r.Use(middleware_authn.Attempt(authenticator))

	r.Route(fmt.Sprintf("/{%s}", request.RepoRefParamName), func(r chi.Router) {
		// resolves the repo and stores in the context
		r.Use(resolve.Repo(repoStore))

		// Write operations (need auth)
		r.Group(func(r chi.Router) {
			// TODO: specific permission for pushing code?
			r.Use(guard.ForRepo(enum.PermissionRepoEdit, false))

			r.Handle("/git-upload-pack", http.HandlerFunc(stubGitHandler))
		})

		// Read operations (only need of it not public)
		r.Group(func(r chi.Router) {
			// middlewares
			r.Use(guard.ForRepo(enum.PermissionRepoView, true))
			// handlers
			r.Post("/git-receive-pack", stubGitHandler)
			r.Get("/info/refs", stubGitHandler)
			r.Get("/HEAD", stubGitHandler)
			r.Get("/objects/info/alternates", stubGitHandler)
			r.Get("/objects/info/http-alternates", stubGitHandler)
			r.Get("/objects/info/packs", stubGitHandler)
			r.Get("/objects/info/{file:[^/]*}", stubGitHandler)
			r.Get("/objects/{head:[0-9a-f]{2}}/{hash:[0-9a-f]{38}}", stubGitHandler)
			r.Get("/objects/pack/pack-{file:[0-9a-f]{40}}.pack", stubGitHandler)
			r.Get("/objects/pack/pack-{file:[0-9a-f]{40}}.idx", stubGitHandler)
		})
	})

	return r
}

func stubGitHandler(w http.ResponseWriter, r *http.Request) {
	rep, _ := request.RepoFrom(r.Context())

	w.WriteHeader(http.StatusTeapot)
	_, _ = w.Write([]byte(fmt.Sprintf(
		"Oooops, seems you hit a major construction site ... \n"+
			"  Repo: '%s' (%s)\n"+
			"  Method: '%s'\n"+
			"  Path: '%s'\n"+
			"  Query: '%s'",
		rep.Name, rep.Path,
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
	)))
}
