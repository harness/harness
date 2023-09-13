// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package manager

import (
	"time"

	"github.com/harness/gitness/internal/pipeline/file"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/drone/drone-go/drone"
	"github.com/drone/runner-go/client"
)

func convertToDroneStage(stage *types.Stage) *drone.Stage {
	return &drone.Stage{
		ID:        stage.ID,
		BuildID:   stage.ExecutionID,
		Number:    int(stage.Number),
		Name:      stage.Name,
		Kind:      stage.Kind,
		Type:      stage.Type,
		Status:    string(stage.Status),
		Error:     stage.Error,
		ErrIgnore: stage.ErrIgnore,
		ExitCode:  stage.ExitCode,
		Machine:   stage.Machine,
		OS:        stage.OS,
		Arch:      stage.Arch,
		Variant:   stage.Variant,
		Kernel:    stage.Kernel,
		Limit:     stage.Limit,
		LimitRepo: stage.LimitRepo,
		Started:   stage.Started / 1e3, // Drone uses Unix() timestamps whereas we use UnixMilli()
		Stopped:   stage.Stopped / 1e3, // Drone uses Unix() timestamps whereas we use UnixMilli()
		Created:   stage.Created / 1e3, // Drone uses Unix() timestamps whereas we use UnixMilli()
		Updated:   stage.Updated / 1e3, // Drone uses Unix() timestamps whereas we use UnixMilli()
		Version:   stage.Version,
		OnSuccess: stage.OnSuccess,
		OnFailure: stage.OnFailure,
		DependsOn: stage.DependsOn,
		Labels:    stage.Labels,
		Steps:     convertToDroneSteps(stage.Steps),
	}
}

func convertToDroneSteps(steps []*types.Step) []*drone.Step {
	droneSteps := make([]*drone.Step, len(steps))
	for i, step := range steps {
		droneSteps[i] = convertToDroneStep(step)
	}
	return droneSteps
}

func convertToDroneStep(step *types.Step) *drone.Step {
	return &drone.Step{
		ID:        step.ID,
		StageID:   step.StageID,
		Number:    int(step.Number),
		Name:      step.Name,
		Status:    string(step.Status),
		Error:     step.Error,
		ErrIgnore: step.ErrIgnore,
		ExitCode:  step.ExitCode,
		Started:   step.Started / 1e3, // Drone uses Unix() timestamps whereas we use UnixMilli()
		Stopped:   step.Stopped / 1e3, // Drone uses Unix() timestamps whereas we use UnixMilli()
		Version:   step.Version,
		DependsOn: step.DependsOn,
		Image:     step.Image,
		Detached:  step.Detached,
		Schema:    step.Schema,
	}
}

func convertFromDroneStep(step *drone.Step) *types.Step {
	return &types.Step{
		ID:        step.ID,
		StageID:   step.StageID,
		Number:    int64(step.Number),
		Name:      step.Name,
		Status:    enum.ParseCIStatus(step.Status),
		Error:     step.Error,
		ErrIgnore: step.ErrIgnore,
		ExitCode:  step.ExitCode,
		Started:   step.Started * 1e3, // Drone uses Unix() timestamps whereas we use UnixMilli()
		Stopped:   step.Stopped * 1e3,
		Version:   step.Version,
		DependsOn: step.DependsOn,
		Image:     step.Image,
		Detached:  step.Detached,
		Schema:    step.Schema,
	}
}

func convertFromDroneSteps(steps []*drone.Step) []*types.Step {
	typesSteps := make([]*types.Step, len(steps))
	for i, step := range steps {
		typesSteps[i] = &types.Step{
			ID:        step.ID,
			StageID:   step.StageID,
			Number:    int64(step.Number),
			Name:      step.Name,
			Status:    enum.ParseCIStatus(step.Status),
			Error:     step.Error,
			ErrIgnore: step.ErrIgnore,
			ExitCode:  step.ExitCode,
			Started:   step.Started * 1e3, // Drone uses Unix() timestamps whereas we use UnixMilli()
			Stopped:   step.Stopped * 1e3, // Drone uses Unix() timestamps whereas we use UnixMilli()
			Version:   step.Version,
			DependsOn: step.DependsOn,
			Image:     step.Image,
			Detached:  step.Detached,
			Schema:    step.Schema,
		}
	}
	return typesSteps
}

func convertFromDroneStage(stage *drone.Stage) *types.Stage {
	return &types.Stage{
		ID:          stage.ID,
		ExecutionID: stage.BuildID,
		Number:      int64(stage.Number),
		Name:        stage.Name,
		Kind:        stage.Kind,
		Type:        stage.Type,
		Status:      enum.ParseCIStatus(stage.Status),
		Error:       stage.Error,
		ErrIgnore:   stage.ErrIgnore,
		ExitCode:    stage.ExitCode,
		Machine:     stage.Machine,
		OS:          stage.OS,
		Arch:        stage.Arch,
		Variant:     stage.Variant,
		Kernel:      stage.Kernel,
		Limit:       stage.Limit,
		LimitRepo:   stage.LimitRepo,
		Started:     stage.Started * 1e3, // Drone uses Unix() timestamps whereas we use UnixMilli()
		Stopped:     stage.Stopped * 1e3, // Drone uses Unix() timestamps whereas we use UnixMilli()
		Version:     stage.Version,
		OnSuccess:   stage.OnSuccess,
		OnFailure:   stage.OnFailure,
		DependsOn:   stage.DependsOn,
		Labels:      stage.Labels,
		Steps:       convertFromDroneSteps(stage.Steps),
	}
}

func convertFromDroneLine(l *drone.Line) *livelog.Line {
	return &livelog.Line{
		Number:    l.Number,
		Message:   l.Message,
		Timestamp: l.Timestamp,
	}
}

func convertToDroneBuild(execution *types.Execution) *drone.Build {
	return &drone.Build{
		ID:           execution.ID,
		RepoID:       execution.RepoID,
		Trigger:      execution.Trigger,
		Number:       execution.Number,
		Parent:       execution.Parent,
		Status:       string(execution.Status),
		Error:        execution.Error,
		Event:        execution.Event,
		Action:       execution.Action,
		Link:         execution.Link,
		Timestamp:    execution.Timestamp,
		Title:        execution.Title,
		Message:      execution.Message,
		Before:       execution.Before,
		After:        execution.After,
		Ref:          execution.Ref,
		Fork:         execution.Fork,
		Source:       execution.Source,
		Target:       execution.Target,
		Author:       execution.Author,
		AuthorName:   execution.AuthorName,
		AuthorEmail:  execution.AuthorEmail,
		AuthorAvatar: execution.AuthorAvatar,
		Sender:       execution.Sender,
		Params:       execution.Params,
		Cron:         execution.Cron,
		Deploy:       execution.Deploy,
		DeployID:     execution.DeployID,
		Debug:        execution.Debug,
		Started:      execution.Started / 1e3,  // Drone uses Unix() timestamps whereas we use UnixMilli()
		Finished:     execution.Finished / 1e3, // Drone uses Unix() timestamps whereas we use UnixMilli()
		Created:      execution.Created / 1e3,  // Drone uses Unix() timestamps whereas we use UnixMilli()
		Updated:      execution.Updated / 1e3,  // Drone uses Unix() timestamps whereas we use UnixMilli()
		Version:      execution.Version,
	}
}

func convertToDroneRepo(repo *types.Repository) *drone.Repo {
	return &drone.Repo{
		ID:      repo.ID,
		UID:     repo.UID,
		UserID:  repo.CreatedBy,
		Name:    repo.UID,
		HTTPURL: repo.GitURL,
		Link:    repo.GitURL,
		Private: !repo.IsPublic,
		Created: repo.Created,
		Updated: repo.Updated,
		Version: repo.Version,
		Branch:  repo.DefaultBranch,
		// TODO: We can get this from configuration once we start populating it.
		// If this is not set drone runner cancels the build.
		Timeout: int64(time.Duration(10 * time.Hour).Seconds()),
	}
}

func convertToDroneFile(file *file.File) *client.File {
	return &client.File{
		Data: file.Data,
	}
}

func convertToDroneSecret(secret *types.Secret) *drone.Secret {
	return &drone.Secret{
		Name: secret.UID,
		Data: secret.Data,
	}
}

func convertToDroneSecrets(secrets []*types.Secret) []*drone.Secret {
	ret := make([]*drone.Secret, len(secrets))
	for i, s := range secrets {
		ret[i] = convertToDroneSecret(s)
	}
	return ret
}
