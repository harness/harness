package router

import (
	"fmt"
	"net/http"

	"github.com/harness/gitness/internal/api"
	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/handler/account"
	handler_space "github.com/harness/gitness/internal/api/handler/space"
	"github.com/harness/gitness/internal/api/handler/system"
	"github.com/harness/gitness/internal/api/handler/user"
	"github.com/harness/gitness/internal/api/middleware/admin"
	middleware_authn "github.com/harness/gitness/internal/api/middleware/authn"
	"github.com/harness/gitness/internal/api/space"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog/hlog"
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
	authenticator authn.Authenticator,
	authorizer authz.Authorizer) (http.Handler, error) {
	// User go-chi router for inner routing
	r := chi.NewRouter()

	// register all routes under mountPath
	r.Route(mountPath, func(r chi.Router) {

		// Apply common api middleware
		r.Use(middleware.NoCache)

		// configure logging middleware.
		// TODO: r.Use(hlog.NewHandler(log.Logger))
		r.Use(hlog.URLHandler("path"))
		r.Use(hlog.MethodHandler("method"))
		r.Use(hlog.RequestIDHandler("request", "Request-Id"))

		r.Route("/v1", func(r chi.Router) {
			setupRoutesV1(r, systemStore, userStore, spaceStore, authenticator, authorizer)
		})
	})

	// Generate list of all path prefixes that expect terminated FQNs
	terminatedFQNPrefixes := []string{
		mountPath + "/v1/spaces",
		mountPath + "/v1/repos",
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// ensure we encode all terminated FQNs.
		req, _ = api.EncodeTerminatedFQNs(req, terminatedFQNPrefixes)

		r.ServeHTTP(w, req)
	}), nil
}

func setupRoutesV1(
	r chi.Router,
	systemStore store.SystemStore,
	userStore store.UserStore,
	spaceStore store.SpaceStore,
	authenticator authn.Authenticator,
	authorizer authz.Authorizer) {

	// Create singleton middlewares for later usage
	auth := middleware_authn.Enforce(authenticator)
	guard := guard.New(spaceStore, authorizer)

	// SPACES
	r.Route("/spaces", func(r chi.Router) {
		// enforce auth
		r.Use(auth)

		// TODO: Handle public spaces?
		r.Route(fmt.Sprintf("/{%s}", space.RefParamName), func(r chi.Router) {
			// space level operations
			r.Get("/", handler_space.HandleFind(guard, spaceStore))
			r.Post("/", handler_space.HandleCreate(guard, spaceStore))
			r.Put("/", handler_space.HandleUpdate(guard))
			r.Delete("/", handler_space.HandleDelete(guard, spaceStore))

			// space sub operations
			r.Get("/spaces", handler_space.HandleList(guard, spaceStore))
		})
	})

	// REPOS
	r.Route("/repos", func(r chi.Router) {
		// enforce auth
		r.Use(auth)

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("List repositories"))
		})

		r.Route("/{rref}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(fmt.Sprintf("Get repository '%s'", chi.URLParam(r, "rref"))))
			})
			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(fmt.Sprintf("Create repository '%s'", chi.URLParam(r, "rref"))))
			})
			r.Put("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(fmt.Sprintf("Update repository '%s'", chi.URLParam(r, "rref"))))
			})
			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(fmt.Sprintf("Delete repository '%s'", chi.URLParam(r, "rref"))))
			})
		})
	})

	// USER - SELF OPERATIONS
	r.Route("/user", func(r chi.Router) {
		r.Use(auth)

		r.Get("/", user.HandleFind())
		r.Patch("/", user.HandleUpdate(userStore))
		r.Post("/token", user.HandleToken(userStore))
	})

	// USERS - ADMIN OPERATIONS
	r.Route("/users", func(r chi.Router) {
		// enforce auth + sytem admin
		r.Use(auth)
		r.Use(admin.Encorce)

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
