// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package router provides http handlers for serving the
// web applicationa and API endpoints.
package router

import (
	"context"
	"net/http"

	"github.com/harness/scm/internal/api/handler/account"
	"github.com/harness/scm/internal/api/handler/executions"
	"github.com/harness/scm/internal/api/handler/pipelines"
	"github.com/harness/scm/internal/api/handler/projects"
	"github.com/harness/scm/internal/api/handler/system"
	"github.com/harness/scm/internal/api/handler/user"
	"github.com/harness/scm/internal/api/handler/users"
	"github.com/harness/scm/internal/api/middleware/access"
	"github.com/harness/scm/internal/api/middleware/address"
	"github.com/harness/scm/internal/api/middleware/token"
	"github.com/harness/scm/internal/api/openapi"
	"github.com/harness/scm/internal/store"
	"github.com/harness/scm/web"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"github.com/swaggest/swgui/v3emb"
	"github.com/unrolled/secure"
)

// empty context
var nocontext = context.Background()

// New returns a new http.Handler that routes traffic
// to the appropriate http.Handlers.
func New(
	executionStore store.ExecutionStore,
	pipelineStore store.PipelineStore,
	userStore store.UserStore,
	systemStore store.SystemStore,
) http.Handler {

	// create the router with caching disabled
	// for API endpoints
	r := chi.NewRouter()

	// create the auth middleware.
	auth := token.Must(userStore)

	// retrieve system configuration in order to
	// retrieve security and cors configuration options.
	config := systemStore.Config(nocontext)

	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.NoCache)
		r.Use(middleware.Recoverer)

		// configure middleware to help ascertain the true
		// server address from the incoming http.Request
		r.Use(
			address.Handler(
				config.Server.Proto,
				config.Server.Host,
			),
		)

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

		r.Route("/v1", func(r chi.Router) {
			// pipeline endpoints
			r.Route("/pipelines", func(r chi.Router) {
				r.Use(auth)
				r.Get("/", pipelines.HandleList(pipelineStore))
				r.Post("/", pipelines.HandleCreate(pipelineStore))

				// pipeline endpoints
				r.Route("/{pipeline}", func(r chi.Router) {
					r.Get("/", pipelines.HandleFind(pipelineStore))
					r.Patch("/", pipelines.HandleUpdate(pipelineStore))
					r.Delete("/", pipelines.HandleDelete(pipelineStore))

					// execution endpoints
					r.Route("/executions", func(r chi.Router) {
						r.Get("/", executions.HandleList(pipelineStore, executionStore))
						r.Post("/", executions.HandleCreate(pipelineStore, executionStore))
						r.Get("/{execution}", executions.HandleFind(pipelineStore, executionStore))
						r.Patch("/{execution}", executions.HandleUpdate(pipelineStore, executionStore))
						r.Delete("/{execution}", executions.HandleDelete(pipelineStore, executionStore))
					})
				})
			})

			// authenticated user endpoints
			r.Route("/user", func(r chi.Router) {
				r.Use(auth)

				r.Get("/", user.HandleFind())
				r.Patch("/", user.HandleUpdate(userStore))
				r.Post("/token", user.HandleToken(userStore))
			})

			// user management endpoints
			r.Route("/users", func(r chi.Router) {
				r.Use(auth)
				r.Use(access.SystemAdmin())

				r.Get("/", users.HandleList(userStore))
				r.Post("/", users.HandleCreate(userStore))
				r.Get("/{user}", users.HandleFind(userStore))
				r.Patch("/{user}", users.HandleUpdate(userStore))
				r.Delete("/{user}", users.HandleDelete(userStore))
			})

			// system management endpoints
			r.Route("/system", func(r chi.Router) {
				r.Get("/health", system.HandleHealth)
				r.Get("/version", system.HandleVersion)
			})

			// user login endpoint
			r.Post("/login", account.HandleLogin(userStore, systemStore))

			// user registration endpoint
			r.Post("/register", account.HandleRegister(userStore, systemStore))

			// openapi specification endpoints
			swagger := openapi.Handler()
			r.Handle("/swagger.json", swagger)
			r.Handle("/swagger.yaml", swagger)

		})

		// harness platform project endpoints
		r.Route("/projects", func(r chi.Router) {
			r.Use(auth)
			r.Get("/{project}", projects.HandleFind())
		})

		// harness platform project endpoints
		r.Route("/user", func(r chi.Router) {
			r.Use(auth)
			r.Get("/currentUser", user.HandleCurrent())
			r.Get("/projects", projects.HandleList())
		})
	})

	// create middleware to enforce security best practices for
	// the user interface. note that theis middleware is only used
	// when serving the user interface (not found handler, below).
	sec := secure.New(
		secure.Options{
			AllowedHosts:          config.Secure.AllowedHosts,
			HostsProxyHeaders:     config.Secure.HostsProxyHeaders,
			SSLRedirect:           config.Secure.SSLRedirect,
			SSLTemporaryRedirect:  config.Secure.SSLTemporaryRedirect,
			SSLHost:               config.Secure.SSLHost,
			SSLProxyHeaders:       config.Secure.SSLProxyHeaders,
			STSSeconds:            config.Secure.STSSeconds,
			STSIncludeSubdomains:  config.Secure.STSIncludeSubdomains,
			STSPreload:            config.Secure.STSPreload,
			ForceSTSHeader:        config.Secure.ForceSTSHeader,
			FrameDeny:             config.Secure.FrameDeny,
			ContentTypeNosniff:    config.Secure.ContentTypeNosniff,
			BrowserXssFilter:      config.Secure.BrowserXSSFilter,
			ContentSecurityPolicy: config.Secure.ContentSecurityPolicy,
			ReferrerPolicy:        config.Secure.ReferrerPolicy,
		},
	)

	// openapi playground endpoints
	swagger := v3emb.NewHandler("API Definition", "/api/v1/swagger.yaml", "/swagger")
	r.With(sec.Handler).Handle("/swagger", swagger)
	r.With(sec.Handler).Handle("/swagger/*", swagger)

	// serve all other routes from the embedded filesystem,
	// which in turn serves the user interface.
	r.With(sec.Handler).NotFound(
		web.Handler(),
	)

	return r
}
