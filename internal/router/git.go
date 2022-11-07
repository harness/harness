// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package router

import (
	"context"
	"fmt"
	"net/http"

	"code.gitea.io/gitea/modules/setting"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/handler/repo"

	"github.com/harness/gitness/internal/api/middleware/accesslog"
	"github.com/harness/gitness/internal/api/middleware/encode"
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

// NewGitHandler returns a new GitHandler.
func NewGitHandler(
	repoStore store.RepoStore,
	authenticator authn.Authenticator,
	client gitrpc.Interface,
) GitHandler {
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

	r.Use(HTTPGitEnabled)

	r.Route(fmt.Sprintf("/{%s}", request.PathParamRepoRef), func(r chi.Router) {
		r.Use(BasicAuth("\".\"", authenticator, repoStore))
		// Write operations (need auth)
		r.Group(func(r chi.Router) {
			// middleware for authz?
			r.Handle("/git-upload-pack", repo.GetUploadPack(client))
		})

		// Read operations (only need of it not public)
		r.Group(func(r chi.Router) {
			// middleware for authz?

			// handlers
			r.Post("/git-receive-pack", repo.PostReceivePack(client))
			r.Get("/info/refs", repo.GetInfoRefs(client))
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

	// wrap router in git path encoder.
	return encode.GitPathBefore(r)
}

func HTTPGitEnabled(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if setting.Repository.DisableHTTPGit {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("Interacting with repositories by HTTP protocol is not allowed"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// BasicAuth implements a simple middleware handler for adding basic http auth to a route.
func BasicAuth(realm string, auth authn.Authenticator, repoStore store.RepoStore) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := hlog.FromRequest(r)
			log.Debug().Msgf("BasicAuth middleware: validate path %v", r.URL.Path)
			repoPath, err := request.GetRepoRefFromPath(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				log.Err(err).Msgf("BasicAuth middleware: bad path %v", r.URL.Path)
				return
			}
			log.Debug().Msgf("BasicAuth middleware: find repo by path %v", r.URL.Path)
			repository, err := repoStore.FindByPath(r.Context(), repoPath)
			if err != nil {
				http.Error(w, fmt.Sprintf("Repo '%s' not found.", repoPath), http.StatusNotFound)
				log.Err(err).Msgf("BasicAuth middleware: repo not found %v", r.URL.Path)
				return
			}

			if !repository.IsPublic {
				log.Debug().Msgf("BasicAuth middleware: repo %v is private", repository.UID)
				_, err = auth.Authenticate(r)
				if err != nil {
					basicAuthFailed(w, realm)
					log.Err(err).Msgf("BasicAuth middleware: authorization failed %v", r.URL.Path)
					return
				}
			}
			log.Debug().Msgf("BasicAuth middleware: serve next with CtxKey %v", repo.CtxRepoKey)
			ctx := context.WithValue(r.Context(), repo.CtxRepoKey, repository)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func basicAuthFailed(w http.ResponseWriter, realm string) {
	w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
	w.WriteHeader(http.StatusUnauthorized)
}

func stubGitHandler(repoStore store.RepoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)

		repoPath, _ := request.GetRepoRefFromPath(r)
		repo, err := repoStore.FindByPath(r.Context(), repoPath)
		if err != nil {
			_, _ = w.Write([]byte(fmt.Sprintf("Repo '%s' not found.", repoPath)))
			return
		}

		_, _ = w.Write([]byte(fmt.Sprintf(
			"Oooops, seems you hit a major construction site ... \n"+
				"  Repo: '%s' (%s)\n"+
				"  Method: '%s'\n"+
				"  Path: '%s'\n"+
				"  Query: '%s'",
			repo.UID, repo.Path,
			r.Method,
			r.URL.Path,
			r.URL.RawQuery,
		)))
	}
}
