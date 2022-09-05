package router

import (
	"fmt"
	"net/http"

	"github.com/harness/gitness/internal/api/guard"
	middleware_authn "github.com/harness/gitness/internal/api/middleware/authn"
	"github.com/harness/gitness/internal/api/middleware/encode"
	"github.com/harness/gitness/internal/api/middleware/repo"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

/*
 * Mounts the GIT Router under mountPath.
 * The handler is wrapped within a layer that handles encoding FQNS.
 */
func newGitHandler(
	mountPath string,
	systemStore store.SystemStore,
	userStore store.UserStore,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	authenticator authn.Authenticator,
	authorizer authz.Authorizer) (http.Handler, error) {

	// User go-chi router for inner routing (restricted to mountPath!)
	r := chi.NewRouter()
	r.Route(mountPath, func(r chi.Router) {

		// Create singleton middlewares for later usage
		guard := guard.New(spaceStore, authorizer)

		// Apply common api middleware
		r.Use(middleware.NoCache)
		r.Use(middleware.Recoverer)

		// configure logging middleware.
		r.Use(hlog.NewHandler(log.Logger))
		r.Use(hlog.URLHandler("path"))
		r.Use(hlog.MethodHandler("method"))
		r.Use(hlog.RequestIDHandler("request", "Request-Id"))

		// for now always attempt auth - enforced per operation
		r.Use(middleware_authn.Attempt(authenticator))

		r.Route(fmt.Sprintf("/{%s}", request.RepoRefParamName), func(r chi.Router) {
			// resolves the repo and stores in the context
			r.Use(repo.Required(repoStore))

			// Operations that require auth (write)
			r.Group(func(r chi.Router) {
				r.Use(guard.EnforceAuthenticated)

				r.Handle("/git-upload-pack", http.HandlerFunc(stubGitHandler))
			})

			// Operations that not always require auth (read)
			r.Group(func(r chi.Router) {
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
	})

	return encode.GitFqnBefore(r.ServeHTTP), nil
}

func stubGitHandler(w http.ResponseWriter, r *http.Request) {
	rep, _ := request.RepoFrom(r.Context())

	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(fmt.Sprintf(
		"Work in progress ... \n"+
			"  Repo: '%s' (%s)\n"+
			"  Method: '%s'\n"+
			"  Path: '%s'\n"+
			"  Query: '%s'",
		rep.DisplayName, rep.Fqn,
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
	)))
}
