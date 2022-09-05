package router

import (
	"fmt"
	"net/http"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/handler/account"
	handler_repo "github.com/harness/gitness/internal/api/handler/repo"
	handler_space "github.com/harness/gitness/internal/api/handler/space"
	"github.com/harness/gitness/internal/api/handler/system"
	"github.com/harness/gitness/internal/api/handler/user"
	middleware_authn "github.com/harness/gitness/internal/api/middleware/authn"
	"github.com/harness/gitness/internal/api/middleware/encode"
	"github.com/harness/gitness/internal/api/middleware/repo"
	"github.com/harness/gitness/internal/api/middleware/space"
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
 * Mounts the Rest API Router under mountPath.
 * The handler is wrapped within a layer that handles encoding terminated FQNs.
 */
func newApiHandler(
	mountPath string,
	systemStore store.SystemStore,
	userStore store.UserStore,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	authenticator authn.Authenticator,
	authorizer authz.Authorizer) (http.Handler, error) {

	config := systemStore.Config(nocontext)

	// User go-chi router for inner routing (restricted to mountPath!)
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

		// configure cors middleware
		cors := cors.New(
			cors.Options{
				AllowedOrigins:   config.Cors.AllowedOrigins,
				AllowedMethods:   config.Cors.AllowedMethods,
				AllowedHeaders:   config.Cors.AllowedHeaders,
				ExposedHeaders:   config.Cors.ExposedHeaders,
				AllowCredentials: config.Cors.AllowCredentials,
				MaxAge:           config.Cors.MaxAge,
			},
		)
		r.Use(cors.Handler)

		// for now always attempt auth - enforced per operation
		r.Use(middleware_authn.Attempt(authenticator))

		r.Route("/v1", func(r chi.Router) {
			setupRoutesV1(r, systemStore, userStore, spaceStore, repoStore, authenticator, authorizer)
		})
	})

	// Generate list of all path prefixes that expect terminated FQNs
	terminatedFQNPrefixes := []string{
		mountPath + "/v1/spaces",
		mountPath + "/v1/repos",
	}

	return encode.TerminatedFqnBefore(terminatedFQNPrefixes, r.ServeHTTP), nil
}

func setupRoutesV1(
	r chi.Router,
	systemStore store.SystemStore,
	userStore store.UserStore,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	authenticator authn.Authenticator,
	authorizer authz.Authorizer) {

	// Create singleton middlewares for later usage
	guard := guard.New(spaceStore, authorizer)

	// SPACES
	r.Route("/spaces", func(r chi.Router) {
		r.Route(fmt.Sprintf("/{%s}", request.SpaceRefParamName), func(r chi.Router) {

			// Create doesn't require space itself to exist
			r.Post("/", handler_space.HandleCreate(guard, spaceStore))

			// Anything else requires the space to exist - group for middleware
			r.Group(func(r chi.Router) {
				// resolves the space and stores in the context
				r.Use(space.Required(spaceStore))

				// space level operations
				r.Get("/", handler_space.HandleFind(guard, spaceStore))
				r.Put("/", handler_space.HandleUpdate(guard))
				r.Delete("/", handler_space.HandleDelete(guard, spaceStore))

				// space sub operations
				r.Get("/spaces", handler_space.HandleList(guard, spaceStore))
				r.Get("/repos", handler_space.HandleListRepos(guard, repoStore))
			})
		})
	})

	// REPOS
	r.Route("/repos", func(r chi.Router) {
		r.Route(fmt.Sprintf("/{%s}", request.RepoRefParamName), func(r chi.Router) {
			// Create doesn't require repo itself to exist
			r.Post("/", handler_repo.HandleCreate(guard, spaceStore, repoStore))

			// Anything else requires the repo to exist - group for middleware
			r.Group(func(r chi.Router) {
				// resolves the repo and stores in the context
				r.Use(repo.Required(repoStore))

				// repo level operations
				r.Get("/", handler_repo.HandleFind(guard, repoStore))
				r.Put("/", handler_repo.HandleUpdate(guard))
				r.Delete("/", handler_repo.HandleDelete(guard, repoStore))
			})
		})
	})

	// USER - SELF OPERATIONS
	r.Route("/user", func(r chi.Router) {
		// enforce user authenticated
		r.Use(guard.EnforceAuthenticated)

		r.Get("/", user.HandleFind())
		r.Patch("/", user.HandleUpdate(userStore))
		r.Post("/token", user.HandleToken(userStore))
	})

	// USERS - ADMIN OPERATIONS
	r.Route("/users", func(r chi.Router) {
		// enforce system admin
		r.Use(guard.EnforceAdmin)

		r.Route("/{user}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(fmt.Sprintf("Get user '%s'", chi.URLParam(r, "rref"))))
			})
			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(fmt.Sprintf("Create user '%s'", chi.URLParam(r, "rref"))))
			})
			r.Put("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(fmt.Sprintf("Update user '%s'", chi.URLParam(r, "rref"))))
			})
			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(fmt.Sprintf("Delete user '%s'", chi.URLParam(r, "rref"))))
			})
		})
	})

	// SYSTEM MGMT ENDPOINTS
	r.Route("/system", func(r chi.Router) {
		r.Get("/health", system.HandleHealth)
		r.Get("/version", system.HandleVersion)
	})

	// USER LOGIN & REGISTRATION
	r.Post("/login", account.HandleLogin(userStore, systemStore))
	r.Post("/register", account.HandleRegister(userStore, systemStore))
}
