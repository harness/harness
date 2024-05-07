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

package manager

import (
	"time"

	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/drone/drone-go/drone"
	"github.com/drone/runner-go/client"
)

func ConvertToDroneStage(stage *types.Stage) *drone.Stage {
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
		Steps:     ConvertToDroneSteps(stage.Steps),
	}
}

func ConvertToDroneSteps(steps []*types.Step) []*drone.Step {
	droneSteps := make([]*drone.Step, len(steps))
	for i, step := range steps {
		droneSteps[i] = ConvertToDroneStep(step)
	}
	return droneSteps
}

func ConvertToDroneStep(step *types.Step) *drone.Step {
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

func ConvertFromDroneStep(step *drone.Step) *types.Step {
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

func ConvertFromDroneSteps(steps []*drone.Step) []*types.Step {
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

func ConvertFromDroneStage(stage *drone.Stage) *types.Stage {
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
		Steps:       ConvertFromDroneSteps(stage.Steps),
	}
}

func ConvertFromDroneLine(l *drone.Line) *livelog.Line {
	return &livelog.Line{
		Number:    l.Number,
		Message:   l.Message,
		Timestamp: l.Timestamp,
	}
}

func ConvertToDroneBuild(execution *types.Execution) *drone.Build {
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

func ConvertToDroneRepo(repo *types.Repository, repoIsPublic bool) *drone.Repo {
	return &drone.Repo{
		ID:        repo.ID,
		Trusted:   true, // as builds are running on user machines, the repo is marked trusted.
		UID:       repo.Identifier,
		UserID:    repo.CreatedBy,
		Namespace: repo.Path,
		Name:      repo.Identifier,
		HTTPURL:   repo.GitURL,
		Link:      repo.GitURL,
		Private:   !repoIsPublic,
		Created:   repo.Created,
		Updated:   repo.Updated,
		Version:   repo.Version,
		Branch:    repo.DefaultBranch,
		// TODO: We can get this from configuration once we start populating it.
		// If this is not set drone runner cancels the build.
		Timeout: int64((10 * time.Hour).Seconds()),
	}
}

func ConvertToDroneFile(file *file.File) *client.File {
	return &client.File{
		Data: file.Data,
	}
}

func ConvertToDroneSecret(secret *types.Secret) *drone.Secret {
	return &drone.Secret{
		Name: secret.Identifier,
		Data: secret.Data,
	}
}

func ConvertToDroneSecrets(secrets []*types.Secret) []*drone.Secret {
	ret := make([]*drone.Secret, len(secrets))
	for i, s := range secrets {
		ret[i] = ConvertToDroneSecret(s)
	}
	return ret
}

func ConvertToDroneNetrc(netrc *Netrc) *drone.Netrc {
	if netrc == nil {
		return nil
	}

	return &drone.Netrc{
		Machine:  netrc.Machine,
		Login:    netrc.Login,
		Password: netrc.Password,
	}
}
