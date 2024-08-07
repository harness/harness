// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package router

import (
	"context"
	"fmt"
	"net/http"

	"github.com/harness/gitness/app/api/controller/check"
	"github.com/harness/gitness/app/api/controller/connector"
	"github.com/harness/gitness/app/api/controller/execution"
	controllergithook "github.com/harness/gitness/app/api/controller/githook"
	"github.com/harness/gitness/app/api/controller/gitspace"
	"github.com/harness/gitness/app/api/controller/infraprovider"
	"github.com/harness/gitness/app/api/controller/keywordsearch"
	"github.com/harness/gitness/app/api/controller/logs"
	"github.com/harness/gitness/app/api/controller/migrate"
	"github.com/harness/gitness/app/api/controller/pipeline"
	"github.com/harness/gitness/app/api/controller/plugin"
	"github.com/harness/gitness/app/api/controller/principal"
	"github.com/harness/gitness/app/api/controller/pullreq"
	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/controller/reposettings"
	"github.com/harness/gitness/app/api/controller/secret"
	"github.com/harness/gitness/app/api/controller/serviceaccount"
	"github.com/harness/gitness/app/api/controller/space"
	"github.com/harness/gitness/app/api/controller/system"
	"github.com/harness/gitness/app/api/controller/template"
	"github.com/harness/gitness/app/api/controller/trigger"
	"github.com/harness/gitness/app/api/controller/upload"
	"github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/api/controller/webhook"
	"github.com/harness/gitness/app/api/handler/account"
	handlercheck "github.com/harness/gitness/app/api/handler/check"
	handlerconnector "github.com/harness/gitness/app/api/handler/connector"
	handlerexecution "github.com/harness/gitness/app/api/handler/execution"
	handlergithook "github.com/harness/gitness/app/api/handler/githook"
	handlergitspace "github.com/harness/gitness/app/api/handler/gitspace"
	handlerinfraProvider "github.com/harness/gitness/app/api/handler/infraprovider"
	handlerkeywordsearch "github.com/harness/gitness/app/api/handler/keywordsearch"
	handlerlogs "github.com/harness/gitness/app/api/handler/logs"
	handlermigrate "github.com/harness/gitness/app/api/handler/migrate"
	handlerpipeline "github.com/harness/gitness/app/api/handler/pipeline"
	handlerplugin "github.com/harness/gitness/app/api/handler/plugin"
	handlerprincipal "github.com/harness/gitness/app/api/handler/principal"
	handlerpullreq "github.com/harness/gitness/app/api/handler/pullreq"
	handlerrepo "github.com/harness/gitness/app/api/handler/repo"
	handlerreposettings "github.com/harness/gitness/app/api/handler/reposettings"
	"github.com/harness/gitness/app/api/handler/resource"
	handlersecret "github.com/harness/gitness/app/api/handler/secret"
	handlerserviceaccount "github.com/harness/gitness/app/api/handler/serviceaccount"
	handlerspace "github.com/harness/gitness/app/api/handler/space"
	handlersystem "github.com/harness/gitness/app/api/handler/system"
	handlertemplate "github.com/harness/gitness/app/api/handler/template"
	handlertrigger "github.com/harness/gitness/app/api/handler/trigger"
	handlerupload "github.com/harness/gitness/app/api/handler/upload"
	handleruser "github.com/harness/gitness/app/api/handler/user"
	"github.com/harness/gitness/app/api/handler/users"
	handlerwebhook "github.com/harness/gitness/app/api/handler/webhook"
	"github.com/harness/gitness/app/api/middleware/address"
	middlewareauthn "github.com/harness/gitness/app/api/middleware/authn"
	"github.com/harness/gitness/app/api/middleware/encode"
	"github.com/harness/gitness/app/api/middleware/logging"
	"github.com/harness/gitness/app/api/middleware/nocache"
	middlewareprincipal "github.com/harness/gitness/app/api/middleware/principal"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/hlog"
)

var (
	// terminatedPathPrefixesAPI is the list of prefixes that will require resolving terminated paths.
	terminatedPathPrefixesAPI = []string{"/v1/spaces/", "/v1/repos/",
		"/v1/secrets/", "/v1/connectors", "/v1/templates/step", "/v1/templates/stage",
		"/v1/gitspaces", "/v1/infraproviders", "/v1/migrate/repos"}
)

// NewAPIHandler returns a new APIHandler.
func NewAPIHandler(
	appCtx context.Context,
	config *types.Config,
	authenticator authn.Authenticator,
	repoCtrl *repo.Controller,
	repoSettingsCtrl *reposettings.Controller,
	executionCtrl *execution.Controller,
	logCtrl *logs.Controller,
	spaceCtrl *space.Controller,
	pipelineCtrl *pipeline.Controller,
	secretCtrl *secret.Controller,
	triggerCtrl *trigger.Controller,
	connectorCtrl *connector.Controller,
	templateCtrl *template.Controller,
	pluginCtrl *plugin.Controller,
	pullreqCtrl *pullreq.Controller,
	webhookCtrl *webhook.Controller,
	githookCtrl *controllergithook.Controller,
	git git.Interface,
	saCtrl *serviceaccount.Controller,
	userCtrl *user.Controller,
	principalCtrl principal.Controller,
	checkCtrl *check.Controller,
	sysCtrl *system.Controller,
	uploadCtrl *upload.Controller,
	searchCtrl *keywordsearch.Controller,
	infraProviderCtrl *infraprovider.Controller,
	migrateCtrl *migrate.Controller,
	gitspaceCtrl *gitspace.Controller,
) http.Handler {
	// Use go-chi router for inner routing.
	r := chi.NewRouter()

	// Apply common api middleware.
	r.Use(nocache.NoCache)
	r.Use(middleware.Recoverer)

	// configure logging middleware.
	r.Use(hlog.URLHandler("http.url"))
	r.Use(hlog.MethodHandler("http.method"))
	r.Use(logging.HLogRequestIDHandler())
	r.Use(logging.HLogAccessLogHandler())
	r.Use(address.Handler("", ""))

	// configure cors middleware
	r.Use(corsHandler(config))

	r.Use(audit.Middleware())

	r.Route("/v1", func(r chi.Router) {
		// special methods that don't require authentication
		setupAccountWithoutAuth(r, userCtrl, sysCtrl, config)
		setupSystem(r, config, sysCtrl)
		setupResources(r)

		r.Group(func(r chi.Router) {
			r.Use(middlewareauthn.Attempt(authenticator))

			setupRoutesV1WithAuth(r, appCtx, config, repoCtrl, repoSettingsCtrl, executionCtrl, triggerCtrl, logCtrl,
				pipelineCtrl, connectorCtrl, templateCtrl, pluginCtrl, secretCtrl, spaceCtrl, pullreqCtrl,
				webhookCtrl, githookCtrl, git, saCtrl, userCtrl, principalCtrl, checkCtrl, uploadCtrl,
				searchCtrl, gitspaceCtrl, infraProviderCtrl, migrateCtrl)
		})
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

// nolint: revive // it's the app context, it shouldn't be the first argument
func setupRoutesV1WithAuth(r chi.Router,
	appCtx context.Context,
	config *types.Config,
	repoCtrl *repo.Controller,
	repoSettingsCtrl *reposettings.Controller,
	executionCtrl *execution.Controller,
	triggerCtrl *trigger.Controller,
	logCtrl *logs.Controller,
	pipelineCtrl *pipeline.Controller,
	connectorCtrl *connector.Controller,
	templateCtrl *template.Controller,
	pluginCtrl *plugin.Controller,
	secretCtrl *secret.Controller,
	spaceCtrl *space.Controller,
	pullreqCtrl *pullreq.Controller,
	webhookCtrl *webhook.Controller,
	githookCtrl *controllergithook.Controller,
	git git.Interface,
	saCtrl *serviceaccount.Controller,
	userCtrl *user.Controller,
	principalCtrl principal.Controller,
	checkCtrl *check.Controller,
	uploadCtrl *upload.Controller,
	searchCtrl *keywordsearch.Controller,
	gitspaceCtrl *gitspace.Controller,
	infraProviderCtrl *infraprovider.Controller,
	migrateCtrl *migrate.Controller,
) {
	setupAccountWithAuth(r, userCtrl, config)
	setupSpaces(r, appCtx, spaceCtrl)
	setupRepos(r, repoCtrl, repoSettingsCtrl, pipelineCtrl, executionCtrl, triggerCtrl,
		logCtrl, pullreqCtrl, webhookCtrl, checkCtrl, uploadCtrl)
	setupConnectors(r, connectorCtrl)
	setupTemplates(r, templateCtrl)
	setupSecrets(r, secretCtrl)
	setupUser(r, userCtrl)
	setupServiceAccounts(r, saCtrl)
	setupPrincipals(r, principalCtrl)
	setupInternal(r, githookCtrl, git)
	setupAdmin(r, userCtrl)
	setupPlugins(r, pluginCtrl)
	setupKeywordSearch(r, searchCtrl)
	setupInfraProviders(r, infraProviderCtrl)
	setupGitspaces(r, gitspaceCtrl)
	setupMigrate(r, migrateCtrl)
}

// nolint: revive // it's the app context, it shouldn't be the first argument
func setupSpaces(
	r chi.Router,
	appCtx context.Context,
	spaceCtrl *space.Controller,

) {
	r.Route("/spaces", func(r chi.Router) {
		// Create takes path and parentId via body, not uri
		r.Post("/", handlerspace.HandleCreate(spaceCtrl))
		r.Post("/import", handlerspace.HandleImport(spaceCtrl))

		r.Route(fmt.Sprintf("/{%s}", request.PathParamSpaceRef), func(r chi.Router) {
			// space operations
			r.Get("/", handlerspace.HandleFind(spaceCtrl))
			r.Patch("/", handlerspace.HandleUpdate(spaceCtrl))
			r.Delete("/", handlerspace.HandleSoftDelete(spaceCtrl))
			r.Post("/restore", handlerspace.HandleRestore(spaceCtrl))
			r.Post("/purge", handlerspace.HandlePurge(spaceCtrl))

			r.Get("/events", handlerspace.HandleEvents(appCtx, spaceCtrl))

			r.Post("/import", handlerspace.HandleImportRepositories(spaceCtrl))
			r.Post("/move", handlerspace.HandleMove(spaceCtrl))
			r.Get("/spaces", handlerspace.HandleListSpaces(spaceCtrl))
			r.Get("/repos", handlerspace.HandleListRepos(spaceCtrl))
			r.Get("/service-accounts", handlerspace.HandleListServiceAccounts(spaceCtrl))
			r.Get("/secrets", handlerspace.HandleListSecrets(spaceCtrl))
			r.Get("/connectors", handlerspace.HandleListConnectors(spaceCtrl))
			r.Get("/templates", handlerspace.HandleListTemplates(spaceCtrl))
			r.Get("/gitspaces", handlerspace.HandleListGitspaces(spaceCtrl))
			r.Post("/export", handlerspace.HandleExport(spaceCtrl))
			r.Get("/export-progress", handlerspace.HandleExportProgress(spaceCtrl))
			r.Post("/public-access", handlerspace.HandleUpdatePublicAccess(spaceCtrl))

			r.Route("/members", func(r chi.Router) {
				r.Get("/", handlerspace.HandleMembershipList(spaceCtrl))
				r.Post("/", handlerspace.HandleMembershipAdd(spaceCtrl))
				r.Route(fmt.Sprintf("/{%s}", request.PathParamUserUID), func(r chi.Router) {
					r.Delete("/", handlerspace.HandleMembershipDelete(spaceCtrl))
					r.Patch("/", handlerspace.HandleMembershipUpdate(spaceCtrl))
				})
			})

			r.Route("/labels", func(r chi.Router) {
				r.Post("/", handlerspace.HandleDefineLabel(spaceCtrl))
				r.Get("/", handlerspace.HandleListLabels(spaceCtrl))
				r.Put("/", handlerspace.HandleSaveLabel(spaceCtrl))

				r.Route(fmt.Sprintf("/{%s}", request.PathParamLabelKey), func(r chi.Router) {
					r.Delete("/", handlerspace.HandleDeleteLabel(spaceCtrl))
					r.Patch("/", handlerspace.HandleUpdateLabel(spaceCtrl))

					r.Route("/values", func(r chi.Router) {
						r.Post("/", handlerspace.HandleDefineLabelValue(spaceCtrl))
						r.Get("/", handlerspace.HandleListLabelValues(spaceCtrl))
						r.Route(fmt.Sprintf("/{%s}", request.PathParamLabelValue), func(r chi.Router) {
							r.Delete("/", handlerspace.HandleDeleteLabelValue(spaceCtrl))
							r.Patch("/", handlerspace.HandleUpdateLabelValue(spaceCtrl))
						})
					})
				})
			})
		})
	})
}

func setupRepos(r chi.Router,
	repoCtrl *repo.Controller,
	repoSettingsCtrl *reposettings.Controller,
	pipelineCtrl *pipeline.Controller,
	executionCtrl *execution.Controller,
	triggerCtrl *trigger.Controller,
	logCtrl *logs.Controller,
	pullreqCtrl *pullreq.Controller,
	webhookCtrl *webhook.Controller,
	checkCtrl *check.Controller,
	uploadCtrl *upload.Controller,
) {
	r.Route("/repos", func(r chi.Router) {
		// Create takes path and parentId via body, not uri
		r.Post("/", handlerrepo.HandleCreate(repoCtrl))
		r.Post("/import", handlerrepo.HandleImport(repoCtrl))
		r.Route(fmt.Sprintf("/{%s}", request.PathParamRepoRef), func(r chi.Router) {
			// repo level operations
			r.Get("/", handlerrepo.HandleFind(repoCtrl))
			r.Patch("/", handlerrepo.HandleUpdate(repoCtrl))
			r.Delete("/", handlerrepo.HandleSoftDelete(repoCtrl))
			r.Post("/purge", handlerrepo.HandlePurge(repoCtrl))
			r.Post("/restore", handlerrepo.HandleRestore(repoCtrl))
			r.Post("/public-access", handlerrepo.HandleUpdatePublicAccess(repoCtrl))

			r.Route("/settings", func(r chi.Router) {
				r.Get("/security", handlerreposettings.HandleSecurityFind(repoSettingsCtrl))
				r.Patch("/security", handlerreposettings.HandleSecurityUpdate(repoSettingsCtrl))
				r.Get("/general", handlerreposettings.HandleGeneralFind(repoSettingsCtrl))
				r.Patch("/general", handlerreposettings.HandleGeneralUpdate(repoSettingsCtrl))
			})

			r.Get("/summary", handlerrepo.HandleSummary(repoCtrl))

			r.Post("/move", handlerrepo.HandleMove(repoCtrl))
			r.Get("/service-accounts", handlerrepo.HandleListServiceAccounts(repoCtrl))

			r.Get("/import-progress", handlerrepo.HandleImportProgress(repoCtrl))

			r.Post("/default-branch", handlerrepo.HandleUpdateDefaultBranch(repoCtrl))

			// content operations
			// NOTE: this allows /content and /content/ to both be valid (without any other tricks.)
			// We don't expect there to be any other operations in that route (as that could overlap with file names)
			r.Route("/content", func(r chi.Router) {
				r.Get("/*", handlerrepo.HandleGetContent(repoCtrl))
			})

			r.Get("/paths", handlerrepo.HandleListPaths(repoCtrl))
			r.Post("/path-details", handlerrepo.HandlePathsDetails(repoCtrl))

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
					r.Get("/diff", handlerrepo.HandleCommitDiff(repoCtrl))
				})
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
				r.Post("/", handlerrepo.HandleCreateCommitTag(repoCtrl))
				r.Delete("/*", handlerrepo.HandleDeleteCommitTag(repoCtrl))
			})

			// diffs
			r.Route("/diff", func(r chi.Router) {
				r.Get("/*", handlerrepo.HandleDiff(repoCtrl))
				r.Post("/*", handlerrepo.HandleDiff(repoCtrl))
			})
			r.Route("/diff-stats", func(r chi.Router) {
				r.Get("/*", handlerrepo.HandleDiffStats(repoCtrl))
			})
			r.Route("/merge-check", func(r chi.Router) {
				r.Post("/*", handlerrepo.HandleMergeCheck(repoCtrl))
			})

			r.Get("/codeowners/validate", handlerrepo.HandleCodeOwnersValidate(repoCtrl))

			r.Get(fmt.Sprintf("/archive/%s", request.PathParamArchiveGitRef), handlerrepo.HandleArchive(repoCtrl))

			SetupPullReq(r, pullreqCtrl)

			SetupWebhook(r, webhookCtrl)

			setupPipelines(r, repoCtrl, pipelineCtrl, executionCtrl, triggerCtrl, logCtrl)

			SetupChecks(r, checkCtrl)

			SetupUploads(r, uploadCtrl)

			SetupRules(r, repoCtrl)

			r.Route("/labels", func(r chi.Router) {
				r.Post("/", handlerrepo.HandleDefineLabel(repoCtrl))
				r.Get("/", handlerrepo.HandleListLabels(repoCtrl))
				r.Put("/", handlerrepo.HandleSaveLabel(repoCtrl))

				r.Route(fmt.Sprintf("/{%s}", request.PathParamLabelKey), func(r chi.Router) {
					r.Delete("/", handlerrepo.HandleDeleteLabel(repoCtrl))
					r.Patch("/", handlerrepo.HandleUpdateLabel(repoCtrl))

					r.Route("/values", func(r chi.Router) {
						r.Post("/", handlerrepo.HandleDefineLabelValue(repoCtrl))
						r.Get("/", handlerrepo.HandleListLabelValues(repoCtrl))
						r.Route(fmt.Sprintf("/{%s}", request.PathParamLabelValue), func(r chi.Router) {
							r.Delete("/", handlerrepo.HandleDeleteLabelValue(repoCtrl))
							r.Patch("/", handlerrepo.HandleUpdateLabelValue(repoCtrl))
						})
					})
				})
			})
		})
	})
}

func SetupUploads(r chi.Router, uploadCtrl *upload.Controller) {
	r.Route("/uploads", func(r chi.Router) {
		r.Post("/", handlerupload.HandleUpload(uploadCtrl))
		r.Get("/*", handlerupload.HandleDownoad(uploadCtrl))
	})
}

func setupPipelines(
	r chi.Router,
	repoCtrl *repo.Controller,
	pipelineCtrl *pipeline.Controller,
	executionCtrl *execution.Controller,
	triggerCtrl *trigger.Controller,
	logCtrl *logs.Controller) {
	r.Route("/pipelines", func(r chi.Router) {
		r.Get("/", handlerrepo.HandleListPipelines(repoCtrl))
		// Create takes path and parentId via body, not uri
		r.Post("/", handlerpipeline.HandleCreate(pipelineCtrl))
		r.Get("/generate", handlerrepo.HandlePipelineGenerate(repoCtrl))
		r.Route(fmt.Sprintf("/{%s}", request.PathParamPipelineIdentifier), func(r chi.Router) {
			r.Get("/", handlerpipeline.HandleFind(pipelineCtrl))
			r.Patch("/", handlerpipeline.HandleUpdate(pipelineCtrl))
			r.Delete("/", handlerpipeline.HandleDelete(pipelineCtrl))
			setupExecutions(r, executionCtrl, logCtrl)
			setupTriggers(r, triggerCtrl)
		})
	})
}

func setupConnectors(
	r chi.Router,
	connectorCtrl *connector.Controller,
) {
	r.Route("/connectors", func(r chi.Router) {
		// Create takes path and parentId via body, not uri
		r.Post("/", handlerconnector.HandleCreate(connectorCtrl))
		r.Route(fmt.Sprintf("/{%s}", request.PathParamConnectorRef), func(r chi.Router) {
			r.Get("/", handlerconnector.HandleFind(connectorCtrl))
			r.Patch("/", handlerconnector.HandleUpdate(connectorCtrl))
			r.Delete("/", handlerconnector.HandleDelete(connectorCtrl))
		})
	})
}

func setupTemplates(
	r chi.Router,
	templateCtrl *template.Controller,
) {
	r.Route("/templates", func(r chi.Router) {
		// Create takes path and parentId via body, not uri
		r.Post("/", handlertemplate.HandleCreate(templateCtrl))
		r.Route(fmt.Sprintf("/{%s}/{%s}", request.PathParamTemplateType, request.PathParamTemplateRef),
			func(r chi.Router) {
				r.Get("/", handlertemplate.HandleFind(templateCtrl))
				r.Patch("/", handlertemplate.HandleUpdate(templateCtrl))
				r.Delete("/", handlertemplate.HandleDelete(templateCtrl))
			})
	})
}

func setupSecrets(r chi.Router, secretCtrl *secret.Controller) {
	r.Route("/secrets", func(r chi.Router) {
		// Create takes path and parentId via body, not uri
		r.Post("/", handlersecret.HandleCreate(secretCtrl))
		r.Route(fmt.Sprintf("/{%s}", request.PathParamSecretRef), func(r chi.Router) {
			r.Get("/", handlersecret.HandleFind(secretCtrl))
			r.Patch("/", handlersecret.HandleUpdate(secretCtrl))
			r.Delete("/", handlersecret.HandleDelete(secretCtrl))
		})
	})
}

func setupPlugins(r chi.Router, pluginCtrl *plugin.Controller) {
	r.Route("/plugins", func(r chi.Router) {
		r.Get("/", handlerplugin.HandleList(pluginCtrl))
	})
}

func setupExecutions(
	r chi.Router,
	executionCtrl *execution.Controller,
	logCtrl *logs.Controller,
) {
	r.Route("/executions", func(r chi.Router) {
		r.Get("/", handlerexecution.HandleList(executionCtrl))
		r.Post("/", handlerexecution.HandleCreate(executionCtrl))
		r.Route(fmt.Sprintf("/{%s}", request.PathParamExecutionNumber), func(r chi.Router) {
			r.Get("/", handlerexecution.HandleFind(executionCtrl))
			r.Post("/cancel", handlerexecution.HandleCancel(executionCtrl))
			r.Delete("/", handlerexecution.HandleDelete(executionCtrl))
			r.Get(
				fmt.Sprintf("/logs/{%s}/{%s}",
					request.PathParamStageNumber,
					request.PathParamStepNumber,
				), handlerlogs.HandleFind(logCtrl))
			// TODO: Decide whether API should be /stream/logs/{}/{} or /logs/{}/{}/stream
			r.Get(
				fmt.Sprintf("/logs/{%s}/{%s}/stream",
					request.PathParamStageNumber,
					request.PathParamStepNumber,
				), handlerlogs.HandleTail(logCtrl))
		})
	})
}

func setupTriggers(
	r chi.Router,
	triggerCtrl *trigger.Controller,
) {
	r.Route("/triggers", func(r chi.Router) {
		r.Get("/", handlertrigger.HandleList(triggerCtrl))
		r.Post("/", handlertrigger.HandleCreate(triggerCtrl))
		r.Route(fmt.Sprintf("/{%s}", request.PathParamTriggerIdentifier), func(r chi.Router) {
			r.Get("/", handlertrigger.HandleFind(triggerCtrl))
			r.Patch("/", handlertrigger.HandleUpdate(triggerCtrl))
			r.Delete("/", handlertrigger.HandleDelete(triggerCtrl))
		})
	})
}

func setupInternal(r chi.Router, githookCtrl *controllergithook.Controller, git git.Interface) {
	r.Route("/internal", func(r chi.Router) {
		SetupGitHooks(r, githookCtrl, git)
	})
}

func SetupGitHooks(r chi.Router, githookCtrl *controllergithook.Controller, git git.Interface) {
	r.Route("/git-hooks", func(r chi.Router) {
		r.Post("/"+githook.HTTPRequestPathPreReceive, handlergithook.HandlePreReceive(githookCtrl, git))
		r.Post("/"+githook.HTTPRequestPathUpdate, handlergithook.HandleUpdate(githookCtrl, git))
		r.Post("/"+githook.HTTPRequestPathPostReceive, handlergithook.HandlePostReceive(githookCtrl, git))
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
				r.Post("/apply-suggestions", handlerpullreq.HandleCommentApplySuggestions(pullreqCtrl))
				r.Route(fmt.Sprintf("/{%s}", request.PathParamPullReqCommentID), func(r chi.Router) {
					r.Patch("/", handlerpullreq.HandleCommentUpdate(pullreqCtrl))
					r.Delete("/", handlerpullreq.HandleCommentDelete(pullreqCtrl))
					r.Put("/status", handlerpullreq.HandleCommentStatus(pullreqCtrl))
				})
			})
			r.Route("/reviewers", func(r chi.Router) {
				r.Get("/", handlerpullreq.HandleReviewerList(pullreqCtrl))
				r.Put("/", handlerpullreq.HandleReviewerAdd(pullreqCtrl))
				r.Route(fmt.Sprintf("/{%s}", request.PathParamReviewerID), func(r chi.Router) {
					r.Delete("/", handlerpullreq.HandleReviewerDelete(pullreqCtrl))
				})
			})
			r.Route("/reviews", func(r chi.Router) {
				r.Post("/", handlerpullreq.HandleReviewSubmit(pullreqCtrl))
			})
			r.Post("/merge", handlerpullreq.HandleMerge(pullreqCtrl))
			r.Get("/commits", handlerpullreq.HandleCommits(pullreqCtrl))
			r.Get("/metadata", handlerpullreq.HandleMetadata(pullreqCtrl))

			r.Route("/file-views", func(r chi.Router) {
				r.Put("/", handlerpullreq.HandleFileViewAdd(pullreqCtrl))
				r.Get("/", handlerpullreq.HandleFileViewList(pullreqCtrl))
				r.Delete("/*", handlerpullreq.HandleFileViewDelete(pullreqCtrl))
			})
			r.Get("/codeowners", handlerpullreq.HandleCodeOwner(pullreqCtrl))
			r.Get("/diff", handlerpullreq.HandleDiff(pullreqCtrl))
			r.Post("/diff", handlerpullreq.HandleDiff(pullreqCtrl))
			r.Get("/checks", handlerpullreq.HandleCheckList(pullreqCtrl))

			r.Route("/labels", func(r chi.Router) {
				r.Put("/", handlerpullreq.HandleAssignLabel(pullreqCtrl))
				r.Get("/", handlerpullreq.HandleListLabels(pullreqCtrl))
				r.Route(fmt.Sprintf("/{%s}", request.PathParamLabelID), func(r chi.Router) {
					r.Delete("/", handlerpullreq.HandleUnassignLabel(pullreqCtrl))
				})
			})
		})
	})
}

func SetupWebhook(r chi.Router, webhookCtrl *webhook.Controller) {
	r.Route("/webhooks", func(r chi.Router) {
		r.Post("/", handlerwebhook.HandleCreate(webhookCtrl))
		r.Get("/", handlerwebhook.HandleList(webhookCtrl))

		r.Route(fmt.Sprintf("/{%s}", request.PathParamWebhookIdentifier), func(r chi.Router) {
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

func SetupChecks(r chi.Router, checkCtrl *check.Controller) {
	r.Route("/checks", func(r chi.Router) {
		r.Get("/recent", handlercheck.HandleCheckListRecent(checkCtrl))
		r.Route(fmt.Sprintf("/commits/{%s}", request.PathParamCommitSHA), func(r chi.Router) {
			r.Put("/", handlercheck.HandleCheckReport(checkCtrl))
			r.Get("/", handlercheck.HandleCheckList(checkCtrl))
		})
	})
}

func SetupRules(r chi.Router, repoCtrl *repo.Controller) {
	r.Route("/rules", func(r chi.Router) {
		r.Post("/", handlerrepo.HandleRuleCreate(repoCtrl))
		r.Get("/", handlerrepo.HandleRuleList(repoCtrl))

		r.Route(fmt.Sprintf("/{%s}", request.PathParamRuleIdentifier), func(r chi.Router) {
			r.Patch("/", handlerrepo.HandleRuleUpdate(repoCtrl))
			r.Delete("/", handlerrepo.HandleRuleDelete(repoCtrl))
			r.Get("/", handlerrepo.HandleRuleFind(repoCtrl))
		})
	})
}

func setupUser(r chi.Router, userCtrl *user.Controller) {
	r.Route("/user", func(r chi.Router) {
		// enforce principal authenticated and it's a user
		r.Use(middlewareprincipal.RestrictTo(enum.PrincipalTypeUser))
		r.Get("/", handleruser.HandleFind(userCtrl))
		r.Patch("/", handleruser.HandleUpdate(userCtrl))
		r.Get("/memberships", handleruser.HandleMembershipSpaces(userCtrl))

		// PAT
		r.Route("/tokens", func(r chi.Router) {
			r.Get("/", handleruser.HandleListTokens(userCtrl, enum.TokenTypePAT))
			r.Post("/", handleruser.HandleCreateAccessToken(userCtrl))

			// per token operations
			r.Route(fmt.Sprintf("/{%s}", request.PathParamTokenIdentifier), func(r chi.Router) {
				r.Delete("/", handleruser.HandleDeleteToken(userCtrl, enum.TokenTypePAT))
			})
		})

		// SESSION TOKENS
		r.Route("/sessions", func(r chi.Router) {
			r.Get("/", handleruser.HandleListTokens(userCtrl, enum.TokenTypeSession))

			// per token operations
			r.Route(fmt.Sprintf("/{%s}", request.PathParamTokenIdentifier), func(r chi.Router) {
				r.Delete("/", handleruser.HandleDeleteToken(userCtrl, enum.TokenTypeSession))
			})
		})

		// Private keys
		r.Route("/keys", func(r chi.Router) {
			r.Get("/", handleruser.HandleListPublicKeys(userCtrl))
			r.Post("/", handleruser.HandleCreatePublicKey(userCtrl))
			r.Delete(fmt.Sprintf("/{%s}", request.PathParamPublicKeyIdentifier),
				handleruser.HandleDeletePublicKey(userCtrl))
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
				r.Route(fmt.Sprintf("/{%s}", request.PathParamTokenIdentifier), func(r chi.Router) {
					r.Delete("/", handlerserviceaccount.HandleDeleteToken(saCtrl))
				})
			})
		})
	})
}

func setupSystem(r chi.Router, config *types.Config, sysCtrl *system.Controller) {
	r.Route("/system", func(r chi.Router) {
		r.Get("/health", handlersystem.HandleHealth)
		r.Get("/version", handlersystem.HandleVersion)
		r.Get("/config", handlersystem.HandleGetConfig(config, sysCtrl))
	})
}

func setupResources(r chi.Router) {
	r.Route("/resources", func(r chi.Router) {
		r.Get("/gitignore", resource.HandleGitIgnores())
		r.Get("/license", resource.HandleLicences())
	})
}

func setupPrincipals(r chi.Router, principalCtrl principal.Controller) {
	r.Route("/principals", func(r chi.Router) {
		r.Get("/", handlerprincipal.HandleList(principalCtrl))
		r.Get(fmt.Sprintf("/{%s}", request.PathParamPrincipalID), handlerprincipal.HandleFind(principalCtrl))
		r.Post("/check-emails", handlerprincipal.HandleCheckExistenceByEmail(principalCtrl))
	})
}

func setupKeywordSearch(r chi.Router, searchCtrl *keywordsearch.Controller) {
	r.Post("/search", handlerkeywordsearch.HandleSearch(searchCtrl))
}

func setupGitspaces(r chi.Router, gitspacesCtrl *gitspace.Controller) {
	r.Route("/gitspaces", func(r chi.Router) {
		r.Post("/lookup-repo", handlergitspace.HandleLookupRepo(gitspacesCtrl))
		r.Post("/", handlergitspace.HandleCreateConfig(gitspacesCtrl))
		r.Get("/", handlergitspace.HandleListAllGitspaces(gitspacesCtrl))
		r.Route(fmt.Sprintf("/{%s}", request.PathParamGitspaceIdentifier), func(r chi.Router) {
			r.Get("/", handlergitspace.HandleFind(gitspacesCtrl))
			r.Post("/actions", handlergitspace.HandleAction(gitspacesCtrl))
			r.Delete("/", handlergitspace.HandleDeleteConfig(gitspacesCtrl))
			r.Patch("/", handlergitspace.HandleUpdateConfig(gitspacesCtrl))
			r.Get("/events", handlergitspace.HandleEvents(gitspacesCtrl))
			r.Get("/logs/stream", handlergitspace.HandleLogsStream(gitspacesCtrl))
		})
	})
}

func setupInfraProviders(r chi.Router, infraProviderCtrl *infraprovider.Controller) {
	r.Route("/infraproviders", func(r chi.Router) {
		r.Post("/", handlerinfraProvider.HandleCreateConfig(infraProviderCtrl))
		r.Route(fmt.Sprintf("/{%s}", request.PathParamInfraProviderConfigIdentifier), func(r chi.Router) {
			r.Get("/", handlerinfraProvider.HandleFind(infraProviderCtrl))
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
				r.Patch("/admin", handleruser.HandleUpdateAdmin(userCtrl))
			})
		})
	})
}

func setupAccountWithoutAuth(
	r chi.Router,
	userCtrl *user.Controller,
	sysCtrl *system.Controller,
	config *types.Config,
) {
	cookieName := config.Token.CookieName
	r.Post("/login", account.HandleLogin(userCtrl, cookieName))
	r.Post("/register", account.HandleRegister(userCtrl, sysCtrl, cookieName))
}

func setupAccountWithAuth(r chi.Router, userCtrl *user.Controller, config *types.Config) {
	cookieName := config.Token.CookieName
	r.Post("/logout", account.HandleLogout(userCtrl, cookieName))
}

func setupMigrate(r chi.Router, migCtrl *migrate.Controller) {
	r.Route("/migrate", func(r chi.Router) {
		r.Route("/repos", func(r chi.Router) {
			r.Post("/", handlermigrate.HandleCreateRepo(migCtrl))
			r.Route(fmt.Sprintf("/{%s}", request.PathParamRepoRef), func(r chi.Router) {
				r.Patch("/update-state", handlermigrate.HandleUpdateRepoState(migCtrl))
				r.Post("/pullreqs", handlermigrate.HandlePullRequests(migCtrl))
				r.Post("/webhooks", handlermigrate.HandleWebhooks(migCtrl))
				r.Post("/rules", handlermigrate.HandleRules(migCtrl))
			})
		})
	})
}
