// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package triggerer

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/harness/gitness/internal/pipeline/file"
	"github.com/harness/gitness/internal/pipeline/scheduler"
	"github.com/harness/gitness/internal/pipeline/triggerer/dag"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/drone/drone-yaml/yaml"
	"github.com/drone/drone-yaml/yaml/linter"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

var _ Triggerer = (*triggerer)(nil)

// Hook represents the payload of a post-commit hook.
type Hook struct {
	Parent       int64              `json:"parent"`
	Trigger      string             `json:"trigger"`
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
	executionStore store.ExecutionStore
	stageStore     store.StageStore
	db             *sqlx.DB
	pipelineStore  store.PipelineStore
	fileService    file.FileService
	scheduler      scheduler.Scheduler
	repoStore      store.RepoStore
}

func New(
	executionStore store.ExecutionStore,
	stageStore store.StageStore,
	pipelineStore store.PipelineStore,
	db *sqlx.DB,
	repoStore store.RepoStore,
	scheduler scheduler.Scheduler,
	fileService file.FileService,
) *triggerer {
	return &triggerer{
		executionStore: executionStore,
		stageStore:     stageStore,
		scheduler:      scheduler,
		db:             db,
		pipelineStore:  pipelineStore,
		fileService:    fileService,
		repoStore:      repoStore,
	}
}

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

	// if base.Event == core.TriggerEventPullRequest {
	// 	if repo.IgnorePulls {
	// 		logger.Infoln("trigger: skipping hook. project ignores pull requests")
	// 		return nil, nil
	// 	}
	// 	if repo.IgnoreForks && !strings.EqualFold(base.Fork, repo.Slug) {
	// 		logger.Infoln("trigger: skipping hook. project ignores forks")
	// 		return nil, nil
	// 	}
	// }

	// user, err := t.users.Find(ctx, repo.UserID)
	// if err != nil {
	// 	logger = logger.WithError(err)
	// 	logger.Warnln("trigger: cannot find repository owner")
	// 	return nil, err
	// }

	// if user.Active == false {
	// 	logger.Infoln("trigger: skipping hook. repository owner is inactive")
	// 	return nil, nil
	// }

	// if the commit message is not included we should
	// make an optional API call to the version control
	// system to augment the available information.
	// if base.Message == "" && base.After != "" {
	// 	commit, err := t.commits.Find(ctx, user, repo.Slug, base.After)
	// 	if err == nil && commit != nil {
	// 		base.Message = commit.Message
	// 		if base.AuthorEmail == "" {
	// 			base.AuthorEmail = commit.Author.Email
	// 		}
	// 		if base.AuthorName == "" {
	// 			base.AuthorName = commit.Author.Name
	// 		}
	// 		if base.AuthorAvatar == "" {
	// 			base.AuthorAvatar = commit.Author.Avatar
	// 		}
	// 	}
	// }

	file, err := t.fileService.Get(ctx, repo, pipeline.ConfigPath, base.After)
	if err != nil {
		log.Error().Err(err).Msg("trigger: could not find yaml")
		return nil, err
	}

	// // this code is temporarily in place to detect and convert
	// // the legacy yaml configuration file to the new format.
	// raw.Data, err = converter.ConvertString(raw.Data, converter.Metadata{
	// 	Filename: repo.Config,
	// 	URL:      repo.Link,
	// 	Ref:      base.Ref,
	// })
	// if err != nil {
	// 	logger = logger.WithError(err)
	// 	logger.Warnln("trigger: cannot convert yaml")
	// 	return t.createExecutionWithError(ctx, repo, base, err.Error())
	// }

	manifest, err := yaml.ParseString(string(file.Data))
	if err != nil {
		log.Warn().Err(err).Msg("trigger: cannot parse yaml")
		return t.createExecutionWithError(ctx, pipeline, base, err.Error())
	}

	// verr := t.validate.Validate(ctx, &core.ValidateArgs{
	// 	User:   user,
	// 	Repo:   repo,
	// 	Execution:  tmpExecution,
	// 	Config: raw,
	// })
	// switch verr {
	// case core.ErrValidatorBlock:
	// 	logger.Debugln("trigger: yaml validation error: block pipeline")
	// case core.ErrValidatorSkip:
	// 	logger.Debugln("trigger: yaml validation error: skip pipeline")
	// 	return nil, nil
	// default:
	// 	if verr != nil {
	// 		logger = logger.WithError(err)
	// 		logger.Warnln("trigger: yaml validation error")
	// 		return t.createExecutionWithError(ctx, repo, base, verr.Error())
	// 	}
	// }

	err = linter.Manifest(manifest, true)
	if err != nil {
		log.Warn().Err(err).Msg("trigger: yaml linting error")
		return t.createExecutionWithError(ctx, pipeline, base, err.Error())
	}

	verified := true
	// if repo.Protected && base.Trigger == core.TriggerHook {
	// 	key := signer.KeyString(repo.Secret)
	// 	val := []byte(raw.Data)
	// 	verified, _ = signer.Verify(val, key)
	// }
	// // if pipeline validation failed with a block error, the
	// // pipeline verification should be set to false, which will
	// // force manual review and approval.
	// if verr == core.ErrValidatorBlock {
	// 	verified = false
	// }

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
		node := dag.Add(pipeline.Name, pipeline.DependsOn...)
		node.Skip = true

		if skipBranch(pipeline, base.Target) {
			log.Info().Str("pipeline", pipeline.Name).Msg("trigger: skipping pipeline, does not match branch")
		} else if skipEvent(pipeline, event) {
			log.Info().Str("pipeline", pipeline.Name).Msg("trigger: skipping pipeline, does not match event")
		} else if skipAction(pipeline, string(base.Action)) {
			log.Info().Str("pipeline", pipeline.Name).Msg("trigger: skipping pipeline, does not match action")
		} else if skipRef(pipeline, base.Ref) {
			log.Info().Str("pipeline", pipeline.Name).Msg("trigger: skipping pipeline, does not match ref")
		} else if skipRepo(pipeline, repo.Path) {
			log.Info().Str("pipeline", pipeline.Name).Msg("trigger: skipping pipeline, does not match repo")
		} else if skipCron(pipeline, base.Cron) {
			log.Info().Str("pipeline", pipeline.Name).Msg("trigger: skipping pipeline, does not match cron job")
		} else {
			matched = append(matched, pipeline)
			node.Skip = false
		}
	}

	if dag.DetectCycles() {
		return t.createExecutionWithError(ctx, pipeline, base, "Error: Dependency cycle detected in Pipeline")
	}

	if len(matched) == 0 {
		log.Info().Msg("trigger: skipping execution, no matching pipelines")
		return nil, nil
	}

	pipeline, err = t.pipelineStore.IncrementSeqNum(ctx, pipeline)
	if err != nil {
		log.Error().Err(err).Msg("trigger: cannot increment execution sequence number")
		return nil, err
	}

	now := time.Now().UnixMilli()

	execution := &types.Execution{
		RepoID:     repo.ID,
		PipelineID: pipeline.ID,
		Trigger:    base.Trigger,
		Number:     pipeline.Seq,
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

	stages := make([]*types.Stage, len(matched))
	for i, match := range matched {
		onSuccess := match.Trigger.Status.Match(enum.CIStatusSuccess)
		onFailure := match.Trigger.Status.Match(enum.CIStatusFailure)
		if len(match.Trigger.Status.Include)+len(match.Trigger.Status.Exclude) == 0 {
			onFailure = false
		}

		now := time.Now().UnixMilli()

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
		if verified == false {
			stage.Status = enum.CIStatusBlocked
		} else if len(stage.DependsOn) == 0 {
			stage.Status = enum.CIStatusPending
		}
		stages[i] = stage
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

	err = t.createExecutionWithStages(ctx, execution, stages)
	if err != nil {
		log.Error().Err(err).Msg("trigger: cannot create execution")
		return nil, err
	}

	// err = t.status.Send(ctx, user, &core.StatusInput{
	// 	Repo:  repo,
	// 	Execution: execution,
	// })
	// if err != nil {
	// 	logger = logger.WithError(err)
	// 	logger.Warnln("trigger: cannot create status")
	// }

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

// createExecutionWithStages writes an execution along with its stages in a single transaction.
func (t *triggerer) createExecutionWithStages(
	ctx context.Context,
	execution *types.Execution,
	stages []*types.Stage,
) error {
	return dbtx.New(t.db).WithTx(ctx, func(ctx context.Context) error {
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
		Number:       pipeline.Seq,
		Parent:       base.Parent,
		Status:       enum.CIStatusError,
		Error:        message,
		Event:        string(base.Action.GetTriggerEvent()),
		Action:       string(base.Action),
		Link:         base.Link,
		Title:        base.Title,
		Message:      base.Message,
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

	return execution, nil
}
