// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package router

import (
	"fmt"
	"net/http"

	"github.com/harness/gitness/internal/api/controller/githook"
	"github.com/harness/gitness/internal/api/controller/principal"
	"github.com/harness/gitness/internal/api/controller/pullreq"
	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/controller/serviceaccount"
	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/internal/api/controller/webhook"
	"github.com/harness/gitness/internal/api/handler/account"
	handlergithook "github.com/harness/gitness/internal/api/handler/githook"
	handlerprincipal "github.com/harness/gitness/internal/api/handler/principal"
	handlerpullreq "github.com/harness/gitness/internal/api/handler/pullreq"
	handlerrepo "github.com/harness/gitness/internal/api/handler/repo"
	"github.com/harness/gitness/internal/api/handler/resource"
	handlerserviceaccount "github.com/harness/gitness/internal/api/handler/serviceaccount"
	handlerspace "github.com/harness/gitness/internal/api/handler/space"
	"github.com/harness/gitness/internal/api/handler/system"
	handleruser "github.com/harness/gitness/internal/api/handler/user"
	"github.com/harness/gitness/internal/api/handler/users"
	handlerwebhook "github.com/harness/gitness/internal/api/handler/webhook"
	middlewareauthn "github.com/harness/gitness/internal/api/middleware/authn"
	"github.com/harness/gitness/internal/api/middleware/encode"
	"github.com/harness/gitness/internal/api/middleware/logging"
	middlewareprincipal "github.com/harness/gitness/internal/api/middleware/principal"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/hlog"
)

// APIHandler is an abstraction of an http handler that handles API calls.
type APIHandler interface {
	http.Handler
}

var (
	// terminatedPathPrefixesAPI is the list of prefixes that will require resolving terminated paths.
	terminatedPathPrefixesAPI = []string{"/v1/spaces/", "/v1/repos/"}
)

// NewAPIHandler returns a new APIHandler.
func NewAPIHandler(
	config *types.Config,
	authenticator authn.Authenticator,
	repoCtrl *repo.Controller,
	spaceCtrl *space.Controller,
	pullreqCtrl *pullreq.Controller,
	webhookCtrl *webhook.Controller,
	githookCtrl *githook.Controller,
	saCtrl *serviceaccount.Controller,
	userCtrl *user.Controller,
	principalCtrl *principal.Controller) APIHandler {
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

	// configure cors middleware
	r.Use(corsHandler(config))

	// for now always attempt auth - enforced per operation.
	r.Use(middlewareauthn.Attempt(authenticator))

	r.Route("/v1", func(r chi.Router) {
		setupRoutesV1(r, repoCtrl, spaceCtrl, pullreqCtrl, webhookCtrl, githookCtrl, saCtrl, userCtrl, principalCtrl)
	})

	// wrap router in terminatedPath encoder.
	return encode.TerminatedPathBefore(terminatedPathPrefixesAPI, r)
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

func setupRoutesV1(r chi.Router,
	repoCtrl *repo.Controller, spaceCtrl *space.Controller,
	pullreqCtrl *pullreq.Controller, webhookCtrl *webhook.Controller, githookCtrl *githook.Controller,
	saCtrl *serviceaccount.Controller, userCtrl *user.Controller, principalCtrl *principal.Controller) {
	setupSpaces(r, spaceCtrl)
	setupRepos(r, repoCtrl, pullreqCtrl, webhookCtrl)
	setupUser(r, userCtrl)
	setupServiceAccounts(r, saCtrl)
	setupPrincipals(r, principalCtrl)
	setupInternal(r, githookCtrl)
	setupAdmin(r, userCtrl)
	setupAccount(r, userCtrl)
	setupSystem(r)
	setupResources(r)
}

func setupSpaces(r chi.Router, spaceCtrl *space.Controller) {
	r.Route("/spaces", func(r chi.Router) {
		// Create takes path and parentId via body, not uri
		r.Post("/", handlerspace.HandleCreate(spaceCtrl))

		r.Route(fmt.Sprintf("/{%s}", request.PathParamSpaceRef), func(r chi.Router) {
			// space operations
			r.Get("/", handlerspace.HandleFind(spaceCtrl))
			r.Patch("/", handlerspace.HandleUpdate(spaceCtrl))
			r.Delete("/", handlerspace.HandleDelete(spaceCtrl))

			r.Post("/move", handlerspace.HandleMove(spaceCtrl))
			r.Get("/spaces", handlerspace.HandleListSpaces(spaceCtrl))
			r.Get("/repos", handlerspace.HandleListRepos(spaceCtrl))
			r.Get("/service-accounts", handlerspace.HandleListServiceAccounts(spaceCtrl))

			// Child collections
			r.Route("/paths", func(r chi.Router) {
				r.Get("/", handlerspace.HandleListPaths(spaceCtrl))
				r.Post("/", handlerspace.HandleCreatePath(spaceCtrl))

				// per path operations
				r.Route(fmt.Sprintf("/{%s}", request.PathParamPathID), func(r chi.Router) {
					r.Delete("/", handlerspace.HandleDeletePath(spaceCtrl))
				})
			})
		})
	})
}

func setupRepos(r chi.Router, repoCtrl *repo.Controller, pullreqCtrl *pullreq.Controller,
	webhookCtrl *webhook.Controller) {
	r.Route("/repos", func(r chi.Router) {
		// Create takes path and parentId via body, not uri
		r.Post("/", handlerrepo.HandleCreate(repoCtrl))
		r.Route(fmt.Sprintf("/{%s}", request.PathParamRepoRef), func(r chi.Router) {
			// repo level operations
			r.Get("/", handlerrepo.HandleFind(repoCtrl))
			r.Patch("/", handlerrepo.HandleUpdate(repoCtrl))
			r.Delete("/", handlerrepo.HandleDelete(repoCtrl))

			r.Post("/move", handlerrepo.HandleMove(repoCtrl))
			r.Get("/service-accounts", handlerrepo.HandleListServiceAccounts(repoCtrl))

			// content operations
			// NOTE: this allows /content and /content/ to both be valid (without any other tricks.)
			// We don't expect there to be any other operations in that route (as that could overlap with file names)
			r.Route("/content", func(r chi.Router) {
				r.Get("/*", handlerrepo.HandleGetContent(repoCtrl))
			})

			r.Route("/blame", func(r chi.Router) {
				r.Get("/*", handlerrepo.HandleBlame(repoCtrl))
			})

			r.Route("/raw", func(r chi.Router) {
				r.Get("/*", handlerrepo.HandleRaw(repoCtrl))
			})

			// commit operations
			r.Route("/commits", func(r chi.Router) {
				r.Get("/", handlerrepo.HandleListCommits(repoCtrl))

				r.Post("/calculate-divergence", handlerrepo.HandleCalculateCommitDivergence(repoCtrl))
				r.Post("/", handlerrepo.HandleCommitFiles(repoCtrl))

				// per commit operations
				r.Route(fmt.Sprintf("/{%s}", request.PathParamCommitSHA), func(r chi.Router) {
					r.Get("/", handlerrepo.HandleGetCommit(repoCtrl))
				})
			})

			r.Route("/commitsV2", func(r chi.Router) {
				r.Get("/", handlerrepo.HandleListCommitsV2(repoCtrl))
			})

			// branch operations
			r.Route("/branches", func(r chi.Router) {
				r.Get("/", handlerrepo.HandleListBranches(repoCtrl))
				r.Post("/", handlerrepo.HandleCreateBranch(repoCtrl))

				// per branch operations (can't be grouped in single route)
				r.Get("/*", handlerrepo.HandleGetBranch(repoCtrl))
				r.Delete("/*", handlerrepo.HandleDeleteBranch(repoCtrl))
			})

			// tags operations
			r.Route("/tags", func(r chi.Router) {
				r.Get("/", handlerrepo.HandleListCommitTags(repoCtrl))
				r.Delete("/*", handlerrepo.HandleDeleteCommitTag(repoCtrl))
			})

			// repo path operations
			r.Route("/paths", func(r chi.Router) {
				r.Get("/", handlerrepo.HandleListPaths(repoCtrl))
				r.Post("/", handlerrepo.HandleCreatePath(repoCtrl))

				// per path operations
				r.Route(fmt.Sprintf("/{%s}", request.PathParamPathID), func(r chi.Router) {
					r.Delete("/", handlerrepo.HandleDeletePath(repoCtrl))
				})
			})

			// diffs
			r.Route("/compare", func(r chi.Router) {
				r.Get("/*", handlerrepo.HandleRawDiff(repoCtrl))
			})
			r.Route("/merge-check", func(r chi.Router) {
				r.Post("/*", handlerrepo.HandleMergeCheck(repoCtrl))
			})
			r.Route("/diff-stats", func(r chi.Router) {
				r.Get("/*", handlerrepo.HandleDiffStats(repoCtrl))
			})

			SetupPullReq(r, pullreqCtrl)

			SetupWebhook(r, webhookCtrl)
		})
	})
}

func setupInternal(r chi.Router, githookCtrl *githook.Controller) {
	r.Route("/internal", func(r chi.Router) {
		SetupGitHooks(r, githookCtrl)
	})
}

func SetupGitHooks(r chi.Router, githookCtrl *githook.Controller) {
	r.Route("/git-hooks", func(r chi.Router) {
		r.Post("/pre-receive", handlergithook.HandlePreReceive(githookCtrl))
		r.Post("/update", handlergithook.HandleUpdate(githookCtrl))
		r.Post("/post-receive", handlergithook.HandlePostReceive(githookCtrl))
	})
}

func SetupPullReq(r chi.Router, pullreqCtrl *pullreq.Controller) {
	r.Route("/pullreq", func(r chi.Router) {
		r.Post("/", handlerpullreq.HandleCreate(pullreqCtrl))
		r.Get("/", handlerpullreq.HandleList(pullreqCtrl))

		r.Route(fmt.Sprintf("/{%s}", request.PathParamPullReqNumber), func(r chi.Router) {
			r.Get("/", handlerpullreq.HandleFind(pullreqCtrl))
			r.Patch("/", handlerpullreq.HandleUpdate(pullreqCtrl))
			r.Post("/state", handlerpullreq.HandleState(pullreqCtrl))
			r.Get("/activities", handlerpullreq.HandleListActivities(pullreqCtrl))
			r.Route("/comments", func(r chi.Router) {
				r.Post("/", handlerpullreq.HandleCommentCreate(pullreqCtrl))
				r.Route(fmt.Sprintf("/{%s}", request.PathParamPullReqCommentID), func(r chi.Router) {
					r.Patch("/", handlerpullreq.HandleCommentUpdate(pullreqCtrl))
					r.Delete("/", handlerpullreq.HandleCommentDelete(pullreqCtrl))
					r.Put("/status", handlerpullreq.HandleCommentStatus(pullreqCtrl))
				})
			})
			r.Route("/reviewers", func(r chi.Router) {
				r.Get("/", handlerpullreq.HandleReviewerList(pullreqCtrl))
				r.Put("/", handlerpullreq.HandleReviewerAdd(pullreqCtrl))
			})
			r.Route("/reviews", func(r chi.Router) {
				r.Post("/", handlerpullreq.HandleReviewSubmit(pullreqCtrl))
			})
			r.Post("/merge", handlerpullreq.HandleMerge(pullreqCtrl))
			r.Get("/diff", handlerpullreq.HandleRawDiff(pullreqCtrl))
			r.Get("/commits", handlerpullreq.HandleCommits(pullreqCtrl))
			r.Get("/metadata", handlerpullreq.HandleMetadata(pullreqCtrl))
		})
	})
}

func SetupWebhook(r chi.Router, webhookCtrl *webhook.Controller) {
	r.Route("/webhooks", func(r chi.Router) {
		r.Post("/", handlerwebhook.HandleCreate(webhookCtrl))
		r.Get("/", handlerwebhook.HandleList(webhookCtrl))

		r.Route(fmt.Sprintf("/{%s}", request.PathParamWebhookID), func(r chi.Router) {
			r.Get("/", handlerwebhook.HandleFind(webhookCtrl))
			r.Patch("/", handlerwebhook.HandleUpdate(webhookCtrl))
			r.Delete("/", handlerwebhook.HandleDelete(webhookCtrl))

			r.Route("/executions", func(r chi.Router) {
				r.Get("/", handlerwebhook.HandleListExecutions(webhookCtrl))

				r.Route(fmt.Sprintf("/{%s}", request.PathParamWebhookExecutionID), func(r chi.Router) {
					r.Get("/", handlerwebhook.HandleFindExecution(webhookCtrl))
					r.Post("/retrigger", handlerwebhook.HandleRetriggerExecution(webhookCtrl))
				})
			})
		})
	})
}

func setupUser(r chi.Router, userCtrl *user.Controller) {
	r.Route("/user", func(r chi.Router) {
		// enforce principial authenticated and it's a user
		r.Use(middlewareprincipal.RestrictTo(enum.PrincipalTypeUser))

		r.Get("/", handleruser.HandleFind(userCtrl))
		r.Patch("/", handleruser.HandleUpdate(userCtrl))

		// PAT
		r.Route("/tokens", func(r chi.Router) {
			r.Get("/", handleruser.HandleListTokens(userCtrl, enum.TokenTypePAT))
			r.Post("/", handleruser.HandleCreateAccessToken(userCtrl))

			// per token operations
			r.Route(fmt.Sprintf("/{%s}", request.PathParamTokenUID), func(r chi.Router) {
				r.Delete("/", handleruser.HandleDeleteToken(userCtrl, enum.TokenTypePAT))
			})
		})

		// SESSION TOKENS
		r.Route("/sessions", func(r chi.Router) {
			r.Get("/", handleruser.HandleListTokens(userCtrl, enum.TokenTypeSession))

			// per token operations
			r.Route(fmt.Sprintf("/{%s}", request.PathParamTokenUID), func(r chi.Router) {
				r.Delete("/", handleruser.HandleDeleteToken(userCtrl, enum.TokenTypeSession))
			})
		})
	})
}

func setupServiceAccounts(r chi.Router, saCtrl *serviceaccount.Controller) {
	r.Route("/service-accounts", func(r chi.Router) {
		// create takes parent information via body
		r.Post("/", handlerserviceaccount.HandleCreate(saCtrl))

		r.Route(fmt.Sprintf("/{%s}", request.PathParamServiceAccountUID), func(r chi.Router) {
			r.Get("/", handlerserviceaccount.HandleFind(saCtrl))
			r.Delete("/", handlerserviceaccount.HandleDelete(saCtrl))

			// SAT
			r.Route("/tokens", func(r chi.Router) {
				r.Get("/", handlerserviceaccount.HandleListTokens(saCtrl))
				r.Post("/", handlerserviceaccount.HandleCreateToken(saCtrl))

				// per token operations
				r.Route(fmt.Sprintf("/{%s}", request.PathParamTokenUID), func(r chi.Router) {
					r.Delete("/", handlerserviceaccount.HandleDeleteToken(saCtrl))
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

func setupResources(r chi.Router) {
	r.Route("/resources", func(r chi.Router) {
		r.Get("/gitignore", resource.HandleGitIgnore())
		r.Get("/license", resource.HandleLicence())
	})
}

func setupPrincipals(r chi.Router, principalCtrl *principal.Controller) {
	r.Route("/principals", func(r chi.Router) {
		r.Route(fmt.Sprintf("/{%s}", request.PathParamPrincipalUID), func(r chi.Router) {
			r.Get("/", handlerprincipal.HandleFindPublic(principalCtrl))
		})
	})
}

func setupAdmin(r chi.Router, userCtrl *user.Controller) {
	r.Route("/admin", func(r chi.Router) {
		r.Use(middlewareprincipal.RestrictToAdmin())
		r.Route("/users", func(r chi.Router) {
			r.Get("/", users.HandleList(userCtrl))
			r.Post("/", users.HandleCreate(userCtrl))

			r.Route(fmt.Sprintf("/{%s}", request.PathParamUserUID), func(r chi.Router) {
				r.Get("/", users.HandleFind(userCtrl))
				r.Patch("/", users.HandleUpdate(userCtrl))
				r.Delete("/", users.HandleDelete(userCtrl))
			})
		})
	})
}

func setupAccount(r chi.Router, userCtrl *user.Controller) {
	r.Post("/login", account.HandleLogin(userCtrl))
	r.Post("/register", account.HandleRegister(userCtrl))
}
