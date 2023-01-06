// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package router

import (
	"fmt"
	"net/http"

	"github.com/harness/gitness/gitrpc"
	handlerrepo "github.com/harness/gitness/internal/api/handler/repo"
	middlewareauthn "github.com/harness/gitness/internal/api/middleware/authn"
	"github.com/harness/gitness/internal/api/middleware/encode"
	"github.com/harness/gitness/internal/api/middleware/logging"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog/hlog"
)

// GitHandler is an abstraction of an http handler that handles git calls.
type GitHandler interface {
	http.Handler
}

// NewGitHandler returns a new GitHandler.
func NewGitHandler(
	config *types.Config,
	urlProvider *url.Provider,
	repoStore store.RepoStore,
	authenticator authn.Authenticator,
	authorizer authz.Authorizer,
	client gitrpc.Interface,
) GitHandler {
	// Use go-chi router for inner routing.
	r := chi.NewRouter()

	// Apply common api middleware.
	r.Use(middleware.NoCache)
	r.Use(middleware.Recoverer)

	// configure logging middleware.
	r.Use(hlog.URLHandler("http.url"))
	r.Use(hlog.MethodHandler("http.method"))
	r.Use(logging.HLogRequestIDHandler())
	r.Use(logging.HLogAccessLogHandler())

	r.Route(fmt.Sprintf("/{%s}", request.PathParamRepoRef), func(r chi.Router) {
		r.Use(middlewareauthn.Attempt(authenticator))

		// smart protocol
		r.Handle("/git-upload-pack", handlerrepo.GetUploadPack(client, urlProvider, repoStore, authorizer))
		r.Post("/git-receive-pack", handlerrepo.PostReceivePack(client, urlProvider, repoStore, authorizer))
		r.Get("/info/refs", handlerrepo.GetInfoRefs(client, repoStore, authorizer))

		// dumb protocol
		r.Get("/HEAD", stubGitHandler(repoStore))
		r.Get("/objects/info/alternates", stubGitHandler(repoStore))
		r.Get("/objects/info/http-alternates", stubGitHandler(repoStore))
		r.Get("/objects/info/packs", stubGitHandler(repoStore))
		r.Get("/objects/info/{file:[^/]*}", stubGitHandler(repoStore))
		r.Get("/objects/{head:[0-9a-f]{2}}/{hash:[0-9a-f]{38}}", stubGitHandler(repoStore))
		r.Get("/objects/pack/pack-{file:[0-9a-f]{40}}.pack", stubGitHandler(repoStore))
		r.Get("/objects/pack/pack-{file:[0-9a-f]{40}}.idx", stubGitHandler(repoStore))
	})

	// wrap router in git path encoder.
	return encode.GitPathBefore(r)
}

func stubGitHandler(repoStore store.RepoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Seems like an asteroid destroyed the ancient git protocol"))
		w.WriteHeader(http.StatusBadGateway)
	}
}
