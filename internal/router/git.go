package router

import (
	"fmt"
	"net/http"

	"github.com/harness/gitness/internal/api/middleware/accesslog"
	middleware_authn "github.com/harness/gitness/internal/api/middleware/authn"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/store"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog/hlog"
)

// GitHandler is an abstraction of an http handler that handles git calls.
type GitHandler interface {
	http.Handler
}

/*
 * NewGitHandler returns a new GitHandler.
 */
func NewGitHandler(
	repoStore store.RepoStore,
	authenticator authn.Authenticator) GitHandler {
	// Use go-chi router for inner routing.
	r := chi.NewRouter()

	// Apply common api middleware.
	r.Use(middleware.NoCache)
	r.Use(middleware.Recoverer)

	// configure logging middleware.
	r.Use(hlog.URLHandler("url"))
	r.Use(hlog.MethodHandler("method"))
	r.Use(hlog.RequestIDHandler("request", "Request-Id"))
	r.Use(accesslog.HlogHandler())

	// for now always attempt auth - enforced per operation.
	r.Use(middleware_authn.Attempt(authenticator))

	r.Route(fmt.Sprintf("/{%s}", request.PathParamRepoRef), func(r chi.Router) {
		// Write operations (need auth)
		r.Group(func(r chi.Router) {
			// middleware for authz?
			r.Handle("/git-upload-pack", stubGitHandler(repoStore))
		})

		// Read operations (only need of it not public)
		r.Group(func(r chi.Router) {
			// middleware for authz?

			// handlers
			r.Post("/git-receive-pack", stubGitHandler(repoStore))
			r.Get("/info/refs", stubGitHandler(repoStore))
			r.Get("/HEAD", stubGitHandler(repoStore))
			r.Get("/objects/info/alternates", stubGitHandler(repoStore))
			r.Get("/objects/info/http-alternates", stubGitHandler(repoStore))
			r.Get("/objects/info/packs", stubGitHandler(repoStore))
			r.Get("/objects/info/{file:[^/]*}", stubGitHandler(repoStore))
			r.Get("/objects/{head:[0-9a-f]{2}}/{hash:[0-9a-f]{38}}", stubGitHandler(repoStore))
			r.Get("/objects/pack/pack-{file:[0-9a-f]{40}}.pack", stubGitHandler(repoStore))
			r.Get("/objects/pack/pack-{file:[0-9a-f]{40}}.idx", stubGitHandler(repoStore))
		})
	})

	return r
}

func stubGitHandler(repoStore store.RepoStore) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		repoPath, _ := request.GetRepoRef(r)
		repo, err := repoStore.FindByPath(r.Context(), repoPath)
		if err != nil {
			_, _ = w.Write([]byte(fmt.Sprintf("Repo '%s' not found.", repoPath)))
			return
		}

		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(fmt.Sprintf(
			"Oooops, seems you hit a major construction site ... \n"+
				"  Repo: '%s' (%s)\n"+
				"  Method: '%s'\n"+
				"  Path: '%s'\n"+
				"  Query: '%s'",
			repo.Name, repo.Path,
			r.Method,
			r.URL.Path,
			r.URL.RawQuery,
		)))
	})
}
