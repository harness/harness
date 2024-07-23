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

package triggerer

import (
	"context"
	"fmt"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/harness/gitness/app/pipeline/checks"
	"github.com/harness/gitness/app/pipeline/converter"
	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/app/pipeline/manager"
	"github.com/harness/gitness/app/pipeline/resolver"
	"github.com/harness/gitness/app/pipeline/scheduler"
	"github.com/harness/gitness/app/pipeline/triggerer/dag"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/drone-runners/drone-runner-docker/engine2/inputs"
	"github.com/drone-runners/drone-runner-docker/engine2/script"
	"github.com/drone/drone-yaml/yaml"
	"github.com/drone/drone-yaml/yaml/linter"
	v1yaml "github.com/drone/spec/dist/go"
	"github.com/drone/spec/dist/go/parse/normalize"
	specresolver "github.com/drone/spec/dist/go/parse/resolver"
	"github.com/rs/zerolog/log"
)

var _ Triggerer = (*triggerer)(nil)

// Hook represents the payload of a post-commit hook.
type Hook struct {
	Parent       int64              `json:"parent"`
	Trigger      string             `json:"trigger"`
	TriggeredBy  int64              `json:"triggered_by"`
	Action       enum.TriggerAction `json:"action"`
	Link         string             `json:"link"`
	Timestamp    int64              `json:"timestamp"`
	Title        string             `json:"title"`
	Message      string             `json:"message"`
	Before       string             `json:"before"`
	After        string             `json:"after"`
	Ref          string             `json:"ref"`
	Fork         string             `json:"fork"`
	Source       string             `json:"source"`
	Target       string             `json:"target"`
	AuthorLogin  string             `json:"author_login"`
	AuthorName   string             `json:"author_name"`
	AuthorEmail  string             `json:"author_email"`
	AuthorAvatar string             `json:"author_avatar"`
	Debug        bool               `json:"debug"`
	Cron         string             `json:"cron"`
	Sender       string             `json:"sender"`
	Params       map[string]string  `json:"params"`
}

// Triggerer is responsible for triggering a Execution from an
// incoming hook (could be manual or webhook). If an execution is skipped a nil value is
// returned.
type Triggerer interface {
	Trigger(ctx context.Context, pipeline *types.Pipeline, hook *Hook) (*types.Execution, error)
}

type triggerer struct {
	executionStore   store.ExecutionStore
	checkStore       store.CheckStore
	stageStore       store.StageStore
	tx               dbtx.Transactor
	pipelineStore    store.PipelineStore
	fileService      file.Service
	converterService converter.Service
	urlProvider      url.Provider
	scheduler        scheduler.Scheduler
	repoStore        store.RepoStore
	templateStore    store.TemplateStore
	pluginStore      store.PluginStore
	publicAccess     publicaccess.Service
}

func New(
	executionStore store.ExecutionStore,
	checkStore store.CheckStore,
	stageStore store.StageStore,
	pipelineStore store.PipelineStore,
	tx dbtx.Transactor,
	repoStore store.RepoStore,
	urlProvider url.Provider,
	scheduler scheduler.Scheduler,
	fileService file.Service,
	converterService converter.Service,
	templateStore store.TemplateStore,
	pluginStore store.PluginStore,
	publicAccess publicaccess.Service,
) Triggerer {
	return &triggerer{
		executionStore:   executionStore,
		checkStore:       checkStore,
		stageStore:       stageStore,
		scheduler:        scheduler,
		urlProvider:      urlProvider,
		tx:               tx,
		pipelineStore:    pipelineStore,
		fileService:      fileService,
		converterService: converterService,
		repoStore:        repoStore,
		templateStore:    templateStore,
		pluginStore:      pluginStore,
		publicAccess:     publicAccess,
	}
}

//nolint:gocognit,gocyclo,cyclop //TODO: Refactor @Vistaar
func (t *triggerer) Trigger(
	ctx context.Context,
	pipeline *types.Pipeline,
	base *Hook,
) (*types.Execution, error) {
	log := log.With().
		Int64("pipeline.id", pipeline.ID).
		Str("trigger.ref", base.Ref).
		Str("trigger.commit", base.After).
		Logger()

	log.Debug().Msg("trigger: received")
	defer func() {
		// taking the paranoid approach to recover from
		// a panic that should absolutely never happen.
		if r := recover(); r != nil {
			log.Error().Msgf("runner: unexpected panic: %s", r)
			debug.PrintStack()
		}
	}()

	event := string(base.Action.GetTriggerEvent())

	repo, err := t.repoStore.Find(ctx, pipeline.RepoID)
	if err != nil {
		log.Error().Err(err).Msg("could not find repo")
		return nil, err
	}

	repoIsPublic, err := t.publicAccess.Get(ctx, enum.PublicResourceTypeRepo, repo.Path)
	if err != nil {
		return nil, fmt.Errorf("could not check if repo is public: %w", err)
	}

	file, err := t.fileService.Get(ctx, repo, pipeline.ConfigPath, base.After)
	if err != nil {
		log.Error().Err(err).Msg("trigger: could not find yaml")
		return nil, err
	}

	now := time.Now().UnixMilli()
	execution := &types.Execution{
		RepoID:     repo.ID,
		PipelineID: pipeline.ID,
		Trigger:    base.Trigger,
		CreatedBy:  base.TriggeredBy,
		Parent:     base.Parent,
		Status:     enum.CIStatusPending,
		Event:      event,
		Action:     string(base.Action),
		Link:       base.Link,
		// Timestamp:    base.Timestamp,
		Title:        trunc(base.Title, 2000),
		Message:      trunc(base.Message, 2000),
		Before:       base.Before,
		After:        base.After,
		Ref:          base.Ref,
		Fork:         base.Fork,
		Source:       base.Source,
		Target:       base.Target,
		Author:       base.AuthorLogin,
		AuthorName:   base.AuthorName,
		AuthorEmail:  base.AuthorEmail,
		AuthorAvatar: base.AuthorAvatar,
		Params:       base.Params,
		Debug:        base.Debug,
		Sender:       base.Sender,
		Cron:         base.Cron,
		Created:      now,
		Updated:      now,
	}

	// For drone, follow the existing path of calculating dependencies, creating a DAG,
	// and creating stages accordingly. For V1 YAML - for now we can just parse the stages
	// and create them sequentially.
	stages := []*types.Stage{}
	//nolint:nestif // refactor if needed
	if !isV1Yaml(file.Data) {
		// Convert from jsonnet/starlark to drone yaml
		args := &converter.ConvertArgs{
			Repo:         repo,
			Pipeline:     pipeline,
			Execution:    execution,
			File:         file,
			RepoIsPublic: repoIsPublic,
		}
		file, err = t.converterService.Convert(ctx, args)
		if err != nil {
			log.Warn().Err(err).Msg("trigger: cannot convert from template")
			return t.createExecutionWithError(ctx, pipeline, base, err.Error())
		}

		manifest, err := yaml.ParseString(string(file.Data))
		if err != nil {
			log.Warn().Err(err).Msg("trigger: cannot parse yaml")
			return t.createExecutionWithError(ctx, pipeline, base, err.Error())
		}

		err = linter.Manifest(manifest, true)
		if err != nil {
			log.Warn().Err(err).Msg("trigger: yaml linting error")
			return t.createExecutionWithError(ctx, pipeline, base, err.Error())
		}

		var matched []*yaml.Pipeline
		var dag = dag.New()
		for _, document := range manifest.Resources {
			pipeline, ok := document.(*yaml.Pipeline)
			if !ok {
				continue
			}
			// TODO add repo
			// TODO add instance
			// TODO add target
			// TODO add ref
			name := pipeline.Name
			if name == "" {
				name = "default"
			}
			node := dag.Add(name, pipeline.DependsOn...)
			node.Skip = true

			switch {
			case skipBranch(pipeline, base.Target):
				log.Info().Str("pipeline", name).Msg("trigger: skipping pipeline, does not match branch")
			case skipEvent(pipeline, event):
				log.Info().Str("pipeline", name).Msg("trigger: skipping pipeline, does not match event")
			case skipAction(pipeline, string(base.Action)):
				log.Info().Str("pipeline", name).Msg("trigger: skipping pipeline, does not match action")
			case skipRef(pipeline, base.Ref):
				log.Info().Str("pipeline", name).Msg("trigger: skipping pipeline, does not match ref")
			case skipRepo(pipeline, repo.Path):
				log.Info().Str("pipeline", name).Msg("trigger: skipping pipeline, does not match repo")
			case skipCron(pipeline, base.Cron):
				log.Info().Str("pipeline", name).Msg("trigger: skipping pipeline, does not match cron job")
			default:
				matched = append(matched, pipeline)
				node.Skip = false
			}
		}

		if dag.DetectCycles() {
			return t.createExecutionWithError(ctx, pipeline, base, "Error: Dependency cycle detected in Pipeline")
		}

		if len(matched) == 0 {
			log.Info().Msg("trigger: skipping execution, no matching pipelines")
			//nolint:nilnil // on purpose
			return nil, nil
		}

		for i, match := range matched {
			onSuccess := match.Trigger.Status.Match(string(enum.CIStatusSuccess))
			onFailure := match.Trigger.Status.Match(string(enum.CIStatusFailure))
			if len(match.Trigger.Status.Include)+len(match.Trigger.Status.Exclude) == 0 {
				onFailure = false
			}

			stage := &types.Stage{
				RepoID:    repo.ID,
				Number:    int64(i + 1),
				Name:      match.Name,
				Kind:      match.Kind,
				Type:      match.Type,
				OS:        match.Platform.OS,
				Arch:      match.Platform.Arch,
				Variant:   match.Platform.Variant,
				Kernel:    match.Platform.Version,
				Limit:     match.Concurrency.Limit,
				Status:    enum.CIStatusWaitingOnDeps,
				DependsOn: match.DependsOn,
				OnSuccess: onSuccess,
				OnFailure: onFailure,
				Labels:    match.Node,
				Created:   now,
				Updated:   now,
			}
			if stage.Kind == "pipeline" && stage.Type == "" {
				stage.Type = "docker"
			}
			if stage.OS == "" {
				stage.OS = "linux"
			}
			if stage.Arch == "" {
				stage.Arch = "amd64"
			}

			if stage.Name == "" {
				stage.Name = "default"
			}
			if len(stage.DependsOn) == 0 {
				stage.Status = enum.CIStatusPending
			}
			stages = append(stages, stage)
		}

		for _, stage := range stages {
			// here we re-work the dependencies for the stage to
			// account for the fact that some steps may be skipped
			// and may otherwise break the dependency chain.
			stage.DependsOn = dag.Dependencies(stage.Name)

			// if the stage is pending dependencies, but those
			// dependencies are skipped, the stage can be executed
			// immediately.
			if stage.Status == enum.CIStatusWaitingOnDeps &&
				len(stage.DependsOn) == 0 {
				stage.Status = enum.CIStatusPending
			}
		}
	} else {
		stages, err = parseV1Stages(
			ctx, file.Data, repo, execution, t.templateStore, t.pluginStore, t.publicAccess)
		if err != nil {
			return nil, fmt.Errorf("could not parse v1 YAML into stages: %w", err)
		}
	}

	// Increment pipeline number using optimistic locking.
	pipeline, err = t.pipelineStore.IncrementSeqNum(ctx, pipeline)
	if err != nil {
		log.Error().Err(err).Msg("trigger: cannot increment execution sequence number")
		return nil, err
	}
	// TODO: this can be made better. We are setting this later since otherwise any parsing failure
	// would lead to an incremented pipeline sequence number.
	execution.Number = pipeline.Seq
	execution.Params = combine(execution.Params, Envs(ctx, repo, pipeline, t.urlProvider))

	err = t.createExecutionWithStages(ctx, execution, stages)
	if err != nil {
		log.Error().Err(err).Msg("trigger: cannot create execution")
		return nil, err
	}

	// try to write to check store. log on failure but don't error out the execution
	err = checks.Write(ctx, t.checkStore, execution, pipeline)
	if err != nil {
		log.Error().Err(err).Msg("trigger: could not write to check store")
	}

	for _, stage := range stages {
		if stage.Status != enum.CIStatusPending {
			continue
		}
		err = t.scheduler.Schedule(ctx, stage)
		if err != nil {
			log.Error().Err(err).Msg("trigger: cannot enqueue execution")
			return nil, err
		}
	}

	return execution, nil
}

func trunc(s string, i int) string {
	runes := []rune(s)
	if len(runes) > i {
		return string(runes[:i])
	}
	return s
}

// parseV1Stages tries to parse the yaml into a list of stages and returns an error
// if we are unable to do so or the yaml contains something unexpected.
// Currently, all the stages will be executed one after the other on completion.
// Once we have depends on in v1, this will be changed to use the DAG.
//
//nolint:gocognit // refactor if needed.
func parseV1Stages(
	ctx context.Context,
	data []byte,
	repo *types.Repository,
	execution *types.Execution,
	templateStore store.TemplateStore,
	pluginStore store.PluginStore,
	publicAccess publicaccess.Service,
) ([]*types.Stage, error) {
	stages := []*types.Stage{}
	// For V1 YAML, just go through the YAML and create stages serially for now
	config, err := v1yaml.ParseBytes(data)
	if err != nil {
		return nil, fmt.Errorf("could not parse v1 yaml: %w", err)
	}

	// Normalize the config to make sure stage names and step names are unique
	err = normalize.Normalize(config)
	if err != nil {
		return nil, fmt.Errorf("could not normalize v1 yaml: %w", err)
	}

	if config.Kind != "pipeline" {
		return nil, fmt.Errorf("cannot support non-pipeline kinds in v1 at the moment: %w", err)
	}

	// get repo public access
	repoIsPublic, err := publicAccess.Get(ctx, enum.PublicResourceTypeRepo, repo.Path)
	if err != nil {
		return nil, fmt.Errorf("could not check repo public access: %w", err)
	}

	inputParams := map[string]interface{}{}
	inputParams["repo"] = inputs.Repo(manager.ConvertToDroneRepo(repo, repoIsPublic))
	inputParams["build"] = inputs.Build(manager.ConvertToDroneBuild(execution))

	var prevStage string

	// expand stage level templates and plugins
	lookupFunc := func(name, kind, typ, version string) (*v1yaml.Config, error) {
		f := resolver.Resolve(ctx, pluginStore, templateStore, repo.ParentID)
		return f(name, kind, typ, version)
	}

	if err := specresolver.Resolve(config, lookupFunc); err != nil {
		return nil, fmt.Errorf("could not resolve yaml plugins/templates: %w", err)
	}

	switch v := config.Spec.(type) {
	case *v1yaml.Pipeline:
		// Expand expressions in strings and matrices
		script.ExpandConfig(config, inputParams)

		for idx, stage := range v.Stages {
			// Only parse CI stages for now
			switch stage.Spec.(type) {
			case *v1yaml.StageCI:
				now := time.Now().UnixMilli()
				var onSuccess, onFailure bool
				onSuccess = true
				if stage.When != nil {
					if when := stage.When.Eval; when != "" {
						// TODO: pass in params for resolution
						onSuccess, onFailure, err = script.EvalWhen(when, inputParams)
						if err != nil {
							return nil, fmt.Errorf("could not resolve when condition for stage: %w", err)
						}
					}
				}

				dependsOn := []string{}
				if prevStage != "" {
					dependsOn = append(dependsOn, prevStage)
				}
				status := enum.CIStatusWaitingOnDeps
				// If the stage has no dependencies, it can be picked up for execution.
				if len(dependsOn) == 0 {
					status = enum.CIStatusPending
				}
				temp := &types.Stage{
					RepoID:    repo.ID,
					Number:    int64(idx + 1),
					Name:      stage.Id, // for v1, ID is the unique identifier per stage
					Created:   now,
					Updated:   now,
					Status:    status,
					OnSuccess: onSuccess,
					OnFailure: onFailure,
					DependsOn: dependsOn,
				}
				prevStage = temp.Name
				stages = append(stages, temp)
			default:
				return nil, fmt.Errorf("only CI and template stages are supported in v1 at the moment")
			}
		}
	default:
		return nil, fmt.Errorf("unknown yaml: %w", err)
	}
	return stages, nil
}

// Checks whether YAML is V1 Yaml or drone Yaml.
func isV1Yaml(data []byte) bool {
	// if we are dealing with the legacy drone yaml, use
	// the legacy drone engine.
	return regexp.MustCompilePOSIX(`^spec:`).Match(data)
}

// createExecutionWithStages writes an execution along with its stages in a single transaction.
func (t *triggerer) createExecutionWithStages(
	ctx context.Context,
	execution *types.Execution,
	stages []*types.Stage,
) error {
	return t.tx.WithTx(ctx, func(ctx context.Context) error {
		err := t.executionStore.Create(ctx, execution)
		if err != nil {
			return err
		}

		for _, stage := range stages {
			stage.ExecutionID = execution.ID
			err := t.stageStore.Create(ctx, stage)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// createExecutionWithError creates an execution with an error message.
func (t *triggerer) createExecutionWithError(
	ctx context.Context,
	pipeline *types.Pipeline,
	base *Hook,
	message string,
) (*types.Execution, error) {
	log := log.With().
		Int64("pipeline.id", pipeline.ID).
		Str("trigger.ref", base.Ref).
		Str("trigger.commit", base.After).
		Logger()

	pipeline, err := t.pipelineStore.IncrementSeqNum(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()

	execution := &types.Execution{
		RepoID:       pipeline.RepoID,
		PipelineID:   pipeline.ID,
		Number:       pipeline.Seq,
		Parent:       base.Parent,
		Status:       enum.CIStatusError,
		Error:        message,
		Event:        string(base.Action.GetTriggerEvent()),
		Action:       string(base.Action),
		Link:         base.Link,
		Title:        base.Title,
		Message:      base.Message,
		CreatedBy:    base.TriggeredBy,
		Before:       base.Before,
		After:        base.After,
		Ref:          base.Ref,
		Fork:         base.Fork,
		Source:       base.Source,
		Target:       base.Target,
		Author:       base.AuthorLogin,
		AuthorName:   base.AuthorName,
		AuthorEmail:  base.AuthorEmail,
		AuthorAvatar: base.AuthorAvatar,
		Debug:        base.Debug,
		Sender:       base.Sender,
		Created:      now,
		Updated:      now,
		Started:      now,
		Finished:     now,
	}

	err = t.executionStore.Create(ctx, execution)
	if err != nil {
		log.Error().Err(err).Msg("trigger: cannot create execution error")
		return nil, err
	}

	// try to write to check store, log on failure
	err = checks.Write(ctx, t.checkStore, execution, pipeline)
	if err != nil {
		log.Error().Err(err).Msg("trigger: failed to update check")
	}

	return execution, nil
}
