package router

import (
	"context"
	"fmt"
	"net/http"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/handler/account"
	handlerrepo "github.com/harness/gitness/internal/api/handler/repo"
	handlerserviceaccount "github.com/harness/gitness/internal/api/handler/serviceaccount"
	handlerspace "github.com/harness/gitness/internal/api/handler/space"
	"github.com/harness/gitness/internal/api/handler/system"
	handleruser "github.com/harness/gitness/internal/api/handler/user"
	"github.com/harness/gitness/internal/api/middleware/accesslog"
	middlewareauthn "github.com/harness/gitness/internal/api/middleware/authn"
	"github.com/harness/gitness/internal/api/middleware/resolve"

	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/hlog"
)

/*
 * newAPIHandler returns a new http handler for handling API calls.
 */
func newAPIHandler(
	systemStore store.SystemStore,
	userStore store.UserStore,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	tokenStore store.TokenStore,
	saStore store.ServiceAccountStore,
	authenticator authn.Authenticator,
	authorizer authz.Authorizer) http.Handler {
	//
	config := systemStore.Config(context.Background())
	g := guard.New(authorizer, spaceStore, repoStore)

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

	// configure cors middleware
	r.Use(corsHandler(config))

	// for now always attempt auth - enforced per operation
	r.Use(middlewareauthn.Attempt(authenticator))

	r.Route("/v1", func(r chi.Router) {
		setupRoutesV1(r, systemStore, userStore, spaceStore, repoStore, tokenStore, saStore, authenticator, g)
	})

	return r
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
	repoStore store.RepoStore, tokenStore store.TokenStore, saStore store.ServiceAccountStore, _ authn.Authenticator,
	guard *guard.Guard) {
	setupSpaces(r, spaceStore, repoStore, saStore, guard)
	setupRepos(r, spaceStore, repoStore, saStore, guard)
	setupUsers(r, userStore, tokenStore, guard)
	setupServiceAccounts(r, saStore, tokenStore, guard)
	setupAdmin(r, userStore, guard)
	setupAccount(r, userStore, systemStore, tokenStore)
	setupSystem(r)
}

func setupSpaces(r chi.Router, spaceStore store.SpaceStore, repoStore store.RepoStore,
	saStore store.ServiceAccountStore, guard *guard.Guard) {
	r.Route("/spaces", func(r chi.Router) {
		// Create takes path and parentId via body, not uri
		r.Post("/", handlerspace.HandleCreate(guard, spaceStore))

		r.Route(fmt.Sprintf("/{%s}", request.SpaceRefParamName), func(r chi.Router) {
			// resolves the space and stores in the context
			r.Use(resolve.Space(spaceStore))

			// space operations
			r.Get("/", handlerspace.HandleFind(guard, spaceStore))
			r.Put("/", handlerspace.HandleUpdate(guard, spaceStore))
			r.Delete("/", handlerspace.HandleDelete(guard, spaceStore))

			r.Post("/move", handlerspace.HandleMove(guard, spaceStore))
			r.Get("/spaces", handlerspace.HandleList(guard, spaceStore))
			r.Get("/repos", handlerspace.HandleListRepos(guard, repoStore))
			r.Get("/serviceAccounts", handlerspace.HandleListServiceAccounts(guard, saStore))

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

func setupRepos(r chi.Router, spaceStore store.SpaceStore, repoStore store.RepoStore, saStore store.ServiceAccountStore,
	guard *guard.Guard) {
	r.Route("/repos", func(r chi.Router) {
		// Create takes path and parentId via body, not uri
		r.Post("/", handlerrepo.HandleCreate(guard, spaceStore, repoStore))

		r.Route(fmt.Sprintf("/{%s}", request.RepoRefParamName), func(r chi.Router) {
			// resolves the repo and stores in the context
			r.Use(resolve.Repo(repoStore))

			// repo level operations
			r.Get("/", handlerrepo.HandleFind(guard, repoStore))
			r.Put("/", handlerrepo.HandleUpdate(guard, repoStore))
			r.Delete("/", handlerrepo.HandleDelete(guard, repoStore))

			r.Post("/move", handlerrepo.HandleMove(guard, repoStore, spaceStore))
			r.Get("/serviceAccounts", handlerrepo.HandleListServiceAccounts(guard, saStore))

			// repo path operations
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

func setupUsers(r chi.Router, userStore store.UserStore, tokenStore store.TokenStore, guard *guard.Guard) {
	r.Route("/user", func(r chi.Router) {
		// enforce principial authenticated and it's a user
		r.Use(guard.EnforceAuthenticated)
		r.Use(resolve.UserFromPrincipal(userStore))

		r.Get("/", handleruser.HandleFind)
		r.Patch("/", handleruser.HandleUpdate(userStore))

		// PAT
		r.Route("/tokens", func(r chi.Router) {
			r.Get("/", handleruser.HandleListPATs(tokenStore))
			r.Post("/", handleruser.HandleCreatePAT(tokenStore))

			// per token operations
			r.Route(fmt.Sprintf("/{%s}", request.PatIDParamName), func(r chi.Router) {
				r.Delete("/", handleruser.HandleDeletePAT(tokenStore))
			})
		})

		// SESSION TOKENS
		r.Route("/sessions", func(r chi.Router) {
			r.Get("/", handleruser.HandleListSessionTokens(tokenStore))

			// per token operations
			r.Route(fmt.Sprintf("/{%s}", request.SessionTokenIDParamName), func(r chi.Router) {
				r.Delete("/", handleruser.HandleDeleteSession(tokenStore))
			})
		})
	})
}

func setupServiceAccounts(r chi.Router, saStore store.ServiceAccountStore, tokenStore store.TokenStore,
	guard *guard.Guard) {
	r.Route("/serviceAccounts", func(r chi.Router) {
		// enfore principal is authenticated
		r.Use(guard.EnforceAuthenticated)

		// create takes parent information via body
		r.Post("/", handlerserviceaccount.HandleCreate(guard, saStore))

		r.Route(fmt.Sprintf("/{%s}", request.ServiceAccountUIDParamName), func(r chi.Router) {
			// resolves the service account and stores it in the context
			r.Use(resolve.ServiceAccount(saStore))

			r.Get("/", handlerserviceaccount.HandleFind(guard))
			r.Delete("/", handlerserviceaccount.HandleDelete(guard, saStore, tokenStore))

			// SAT
			r.Route("/tokens", func(r chi.Router) {
				r.Get("/", handlerserviceaccount.HandleListSATs(guard, tokenStore))
				r.Post("/", handlerserviceaccount.HandleCreateSAT(guard, tokenStore))

				// per token operations
				r.Route(fmt.Sprintf("/{%s}", request.SatIDParamName), func(r chi.Router) {
					r.Delete("/", handlerserviceaccount.HandleDeleteSAT(guard, tokenStore))
				})
			})
		})
	})
}

func setupSystem(r chi.Router) {
	r.Route("/system", func(r chi.Router) {
		r.Get("/health", system.HandleHealth)
		r.Get("/version", system.HandleVersion)
	})
}

func setupAdmin(r chi.Router, userStore store.UserStore, guard *guard.Guard) {
	r.Route("/users", func(r chi.Router) {
		// enforce system admin
		r.Use(guard.EnforceAdmin)

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(fmt.Sprintf("Create user '%s'", chi.URLParam(r, "rref"))))
		})

		r.Route(fmt.Sprintf("/{%s}", request.UserUIDParamName), func(r chi.Router) {
			// resolves the user and stores it in the context
			resolve.User(userStore)

			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(fmt.Sprintf("Get user '%s'", chi.URLParam(r, "rref"))))
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

func setupAccount(r chi.Router, userStore store.UserStore, systemStore store.SystemStore,
	tokenStore store.TokenStore) {
	r.Post("/login", account.HandleLogin(userStore, systemStore, tokenStore))
	r.Post("/register", account.HandleRegister(userStore, systemStore, tokenStore))
}
