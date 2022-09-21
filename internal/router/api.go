package router

import (
	"context"
	"fmt"
	"net/http"

	"github.com/harness/gitness/internal/api/handler/account"
	handlerrepo "github.com/harness/gitness/internal/api/handler/repo"
	handlerspace "github.com/harness/gitness/internal/api/handler/space"
	"github.com/harness/gitness/internal/api/handler/system"
	"github.com/harness/gitness/internal/api/handler/user"
	"github.com/harness/gitness/internal/api/middleware/repo"
	"github.com/harness/gitness/internal/api/middleware/space"

	"github.com/harness/gitness/types"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/middleware/accesslog"
	middlewareauthn "github.com/harness/gitness/internal/api/middleware/authn"
	"github.com/harness/gitness/internal/api/middleware/encode"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

/*
 * Mounts the Rest API Router under mountPath (path has to end with ).
 * The handler is wrapped within a layer that handles encoding terminated Paths.
 */
func newAPIHandler(
	mountPath string,
	systemStore store.SystemStore,
	userStore store.UserStore,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	authenticator authn.Authenticator,
	authorizer authz.Authorizer) http.Handler {
	//
	config := systemStore.Config(context.Background())
	g := guard.New(authorizer)

	// Use go-chi router for inner routing (restricted to mountPath!)
	r := chi.NewRouter()
	r.Route(mountPath, func(r chi.Router) {
		// Apply common api middleware
		r.Use(middleware.NoCache)
		r.Use(middleware.Recoverer)

		// configure logging middleware.
		r.Use(hlog.NewHandler(log.Logger))
		r.Use(hlog.URLHandler("path"))
		r.Use(hlog.MethodHandler("method"))
		r.Use(hlog.RequestIDHandler("request", "Request-Id"))
		r.Use(accesslog.HlogHandler())

		// configure cors middleware
		r.Use(corsHandler(config))

		// for now always attempt auth - enforced per operation
		r.Use(middlewareauthn.Attempt(authenticator))

		r.Route("/v1", func(r chi.Router) {
			setupRoutesV1(r, systemStore, userStore, spaceStore, repoStore, authenticator, g)
		})
	})

	// Generate list of all path prefixes that expect terminated Paths
	terminatedPathPrefixes := []string{
		mountPath + "/v1/spaces",
		mountPath + "/v1/repos",
	}

	return encode.TerminatedPathBefore(terminatedPathPrefixes, r.ServeHTTP)
}

func corsHandler(config *types.Config) func(http.Handler) http.Handler {
	return cors.New(
		cors.Options{
			AllowedOrigins:   config.Cors.AllowedOrigins,
			AllowedMethods:   config.Cors.AllowedMethods,
			AllowedHeaders:   config.Cors.AllowedHeaders,
			ExposedHeaders:   config.Cors.ExposedHeaders,
			AllowCredentials: config.Cors.AllowCredentials,
			MaxAge:           config.Cors.MaxAge,
		},
	).Handler
}

func setupRoutesV1(r chi.Router, systemStore store.SystemStore, userStore store.UserStore, spaceStore store.SpaceStore,
	repoStore store.RepoStore, _ authn.Authenticator, guard *guard.Guard) {
	setupSpaces(r, spaceStore, repoStore, guard)
	setupRepos(r, spaceStore, repoStore, guard)
	setupUsers(r, userStore, guard)
	setupAdmin(r, guard)
	setupAuth(r, userStore, systemStore)
	setupSystem(r)
}

func setupSpaces(r chi.Router, spaceStore store.SpaceStore, repoStore store.RepoStore, guard *guard.Guard) {
	r.Route("/spaces", func(r chi.Router) {
		// Create takes path and parentId via body, not uri
		r.Post("/", handlerspace.HandleCreate(guard, spaceStore))

		r.Route(fmt.Sprintf("/{%s}", request.SpaceRefParamName), func(r chi.Router) {
			// resolves the space and stores in the context
			r.Use(space.Required(spaceStore))

			// space operations
			r.Get("/", handlerspace.HandleFind(guard, spaceStore))
			r.Put("/", handlerspace.HandleUpdate(guard, spaceStore))
			r.Delete("/", handlerspace.HandleDelete(guard, spaceStore))

			r.Post("/move", handlerspace.HandleMove(guard, spaceStore))
			r.Get("/spaces", handlerspace.HandleList(guard, spaceStore))
			r.Get("/repos", handlerspace.HandleListRepos(guard, repoStore))

			// Child collections
			r.Route("/paths", func(r chi.Router) {
				r.Get("/", handlerspace.HandleListPaths(guard, spaceStore))
				r.Post("/", handlerspace.HandleCreatePath(guard, spaceStore))

				// per path operations
				r.Route(fmt.Sprintf("/{%s}", request.PathIDParamName), func(r chi.Router) {
					r.Delete("/", handlerspace.HandleDeletePath(guard, spaceStore))
				})
			})
		})
	})
}

func setupRepos(r chi.Router, spaceStore store.SpaceStore, repoStore store.RepoStore, guard *guard.Guard) {
	r.Route("/repos", func(r chi.Router) {
		// Create takes path and parentId via body, not uri
		r.Post("/", handlerrepo.HandleCreate(guard, spaceStore, repoStore))

		r.Route(fmt.Sprintf("/{%s}", request.RepoRefParamName), func(r chi.Router) {
			// resolves the repo and stores in the context
			r.Use(repo.Required(repoStore))

			// repo level operations
			r.Get("/", handlerrepo.HandleFind(guard, repoStore))
			r.Put("/", handlerrepo.HandleUpdate(guard, repoStore))
			r.Delete("/", handlerrepo.HandleDelete(guard, repoStore))

			r.Post("/move", handlerrepo.HandleMove(guard, repoStore, spaceStore))

			// Child collections
			r.Route("/paths", func(r chi.Router) {
				r.Get("/", handlerrepo.HandleListPaths(guard, repoStore))
				r.Post("/", handlerrepo.HandleCreatePath(guard, repoStore))

				// per path operations
				r.Route(fmt.Sprintf("/{%s}", request.PathIDParamName), func(r chi.Router) {
					r.Delete("/", handlerrepo.HandleDeletePath(guard, repoStore))
				})
			})
		})
	})
}

func setupUsers(r chi.Router, userStore store.UserStore, guard *guard.Guard) {
	r.Route("/user", func(r chi.Router) {
		// enforce user authenticated
		r.Use(guard.EnforceAuthenticated)

		r.Get("/", user.HandleFind())
		r.Patch("/", user.HandleUpdate(userStore))
		r.Post("/token", user.HandleToken(userStore))
	})
}

func setupSystem(r chi.Router) {
	r.Route("/system", func(r chi.Router) {
		r.Get("/health", system.HandleHealth)
		r.Get("/version", system.HandleVersion)
	})
}

func setupAdmin(r chi.Router, guard *guard.Guard) {
	r.Route("/users", func(r chi.Router) {
		// enforce system admin
		r.Use(guard.EnforceAdmin)

		r.Route("/{user}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(fmt.Sprintf("Get user '%s'", chi.URLParam(r, "rref"))))
			})
			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(fmt.Sprintf("Create user '%s'", chi.URLParam(r, "rref"))))
			})
			r.Put("/", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(fmt.Sprintf("Update user '%s'", chi.URLParam(r, "rref"))))
			})
			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(fmt.Sprintf("Delete user '%s'", chi.URLParam(r, "rref"))))
			})
		})
	})
}

func setupAuth(r chi.Router, userStore store.UserStore, systemStore store.SystemStore) {
	r.Post("/login", account.HandleLogin(userStore, systemStore))
	r.Post("/register", account.HandleRegister(userStore, systemStore))
}
