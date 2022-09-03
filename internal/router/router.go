// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package router provides http handlers for serving the
// web applicationa and API endpoints.
package router

import (
	"context"
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
)

const (
	restMount = "/api/"
)

// empty context
var nocontext = context.Background()

type Router struct {
	api http.Handler
	git http.Handler
	web http.Handler
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	/*
	 * 1. GIT
	 *
	 * All Git originating traffic starts with "/space1/space2/repo.git".
	 * This can collide with other API endpoints and thus has to be checked first.
	 * To avoid any false positives, we use the user-agent header to identify git agents.
	 */
	a := req.Header.Get("user-agent")
	if strings.HasPrefix(a, "git/") {
		r.git.ServeHTTP(w, req)
		return
	}

	/*
	 * 2. REST API
	 *
	 * All Rest API calls start with "/api/", and thus can be uniquely identified.
	 * Note: This assumes that we are blocking "api" as a space name!
	 */
	p := req.URL.Path
	if strings.HasPrefix(p, restMount) {
		r.api.ServeHTTP(w, req)
		return
	}

	/*
	 * 3. WEB
	 *
	 * Everything else will be routed to web (or return 404)
	 */
	r.web.ServeHTTP(w, req)
}

// New returns a new http.Handler that routes traffic
// to the appropriate http.Handlers.
func New(
	systemStore store.SystemStore,
	userStore store.UserStore,
	spaceStore store.SpaceStore,
	authenticator authn.Authenticator,
	authorizer authz.Authorizer,
) (http.Handler, error) {

	// config := systemStore.Config(nocontext)

	rest, err := newApiHandler("/api", systemStore, userStore, spaceStore, authenticator, authorizer)
	if err != nil {
		return nil, err
	}

	return &Router{
		api: rest,
	}, nil
	// create the auth middleware.
	// auth := token.Must(userStore)

	// retrieve system configuration in order to
	// retrieve security and cors configuration options.

	// r.Route("/api", func(r chi.Router) {
	// 	r.Use(middleware.NoCache)
	// 	r.Use(middleware.Recoverer)

	// 	// configure middleware to help ascertain the true
	// 	// server address from the incoming http.Request
	// 	r.Use(
	// 		address.Handler(
	// 			config.Server.Proto,
	// 			config.Server.Host,
	// 		),
	// 	)

	// 	// configure logging middleware.
	// 	r.Use(hlog.NewHandler(log.Logger))
	// 	r.Use(hlog.URLHandler("path"))
	// 	r.Use(hlog.MethodHandler("method"))
	// 	r.Use(hlog.RequestIDHandler("request", "Request-Id"))

	// 	// configure cors middleware
	// 	cors := cors.New(
	// 		cors.Options{
	// 			AllowedOrigins:   config.Cors.AllowedOrigins,
	// 			AllowedMethods:   config.Cors.AllowedMethods,
	// 			AllowedHeaders:   config.Cors.AllowedHeaders,
	// 			ExposedHeaders:   config.Cors.ExposedHeaders,
	// 			AllowCredentials: config.Cors.AllowCredentials,
	// 			MaxAge:           config.Cors.MaxAge,
	// 		},
	// 	)
	// 	r.Use(cors.Handler)

	// 	r.Route("/v1", func(r chi.Router) {

	// 		// authenticated user endpoints
	// 		r.Route("/user", func(r chi.Router) {
	// 			r.Use(auth)

	// 			r.Get("/", user.HandleFind())
	// 			r.Patch("/", user.HandleUpdate(userStore))
	// 			r.Post("/token", user.HandleToken(userStore))
	// 		})

	// 		// user management endpoints
	// 		r.Route("/users", func(r chi.Router) {
	// 			r.Use(auth)
	// 			r.Use(access.SystemAdmin())

	// 			r.Get("/", users.HandleList(userStore))
	// 			r.Post("/", users.HandleCreate(userStore))
	// 			r.Get("/{user}", users.HandleFind(userStore))
	// 			r.Patch("/{user}", users.HandleUpdate(userStore))
	// 			r.Delete("/{user}", users.HandleDelete(userStore))
	// 		})

	// 		// system management endpoints
	// 		r.Route("/system", func(r chi.Router) {
	// 			r.Get("/health", system.HandleHealth)
	// 			r.Get("/version", system.HandleVersion)
	// 		})

	// 		// user login endpoint
	// 		r.Post("/login", account.HandleLogin(userStore, systemStore))

	// 		// user registration endpoint
	// 		r.Post("/register", account.HandleRegister(userStore, systemStore))

	// 		// openapi specification endpoints
	// 		swagger := openapi.Handler()
	// 		r.Handle("/swagger.json", swagger)
	// 		r.Handle("/swagger.yaml", swagger)

	// 	})

	// 	// harness platform project endpoints
	// 	r.Route("/user", func(r chi.Router) {
	// 		r.Use(auth)
	// 		r.Get("/currentUser", user.HandleCurrent())
	// 	})
	// })

	// // create middleware to enforce security best practices for
	// // the user interface. note that theis middleware is only used
	// // when serving the user interface (not found handler, below).
	// sec := secure.New(
	// 	secure.Options{
	// 		AllowedHosts:          config.Secure.AllowedHosts,
	// 		HostsProxyHeaders:     config.Secure.HostsProxyHeaders,
	// 		SSLRedirect:           config.Secure.SSLRedirect,
	// 		SSLTemporaryRedirect:  config.Secure.SSLTemporaryRedirect,
	// 		SSLHost:               config.Secure.SSLHost,
	// 		SSLProxyHeaders:       config.Secure.SSLProxyHeaders,
	// 		STSSeconds:            config.Secure.STSSeconds,
	// 		STSIncludeSubdomains:  config.Secure.STSIncludeSubdomains,
	// 		STSPreload:            config.Secure.STSPreload,
	// 		ForceSTSHeader:        config.Secure.ForceSTSHeader,
	// 		FrameDeny:             config.Secure.FrameDeny,
	// 		ContentTypeNosniff:    config.Secure.ContentTypeNosniff,
	// 		BrowserXssFilter:      config.Secure.BrowserXSSFilter,
	// 		ContentSecurityPolicy: config.Secure.ContentSecurityPolicy,
	// 		ReferrerPolicy:        config.Secure.ReferrerPolicy,
	// 	},
	// )

	// // openapi playground endpoints
	// swagger := v3emb.NewHandler("API Definition", "/api/v1/swagger.yaml", "/swagger")
	// r.With(sec.Handler).Handle("/swagger", swagger)
	// r.With(sec.Handler).Handle("/swagger/*", swagger)

	// // serve all other routes from the embedded filesystem,
	// // which in turn serves the user interface.
	// r.With(sec.Handler).NotFound(
	// 	web.Handler(),
	// )

	// return r
}
