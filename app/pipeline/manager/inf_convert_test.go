// Copyright 2026 Harness, Inc.
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
	"testing"

	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/drone/drone-go/drone"
	"github.com/stretchr/testify/require"
)

func TestInfConvert_StageRoundTrip(t *testing.T) {
	stage := &types.Stage{
		ID: 1, ExecutionID: 2, Number: 3, Name: "build", Kind: "pipeline", Type: "docker",
		Status: enum.CIStatusRunning, Error: "e", ErrIgnore: true, ExitCode: 4, Machine: "m",
		OS: "linux", Arch: "amd64", Variant: "v8", Kernel: "5", Limit: 6, LimitRepo: 7,
		Started: 8000, Stopped: 9000, Created: 10000, Updated: 11000, Version: 12,
		OnSuccess: true, OnFailure: true, DependsOn: []string{"a"}, Labels: map[string]string{"k": "v"},
		Steps: []*types.Step{{
			ID: 20, StageID: 1, Number: 1, Name: "step", Status: enum.CIStatusSuccess, Error: "se",
			ErrIgnore: true, ExitCode: 1, Started: 1000, Stopped: 2000, Version: 3,
			DependsOn: []string{"x"}, Image: "alpine", Detached: true, Schema: "schema",
		}},
	}

	d := ConvertToDroneStage(stage)
	require.EqualValues(t, 8, d.Started, "unix milli to unix seconds")
	require.EqualValues(t, 9, d.Stopped)
	require.EqualValues(t, 10, d.Created)
	require.EqualValues(t, 11, d.Updated)
	require.Equal(t, "running", d.Status)
	require.EqualValues(t, 2, d.BuildID)
	require.Len(t, d.Steps, 1)
	require.EqualValues(t, 1, d.Steps[0].Started)

	back := ConvertFromDroneStage(d)
	// created / updated are not mapped back; everything else round-trips.
	expected := *stage
	expected.Created = 0
	expected.Updated = 0
	require.Equal(t, expected, *back)
}

func TestInfConvert_StepHelpers(t *testing.T) {
	steps := []*types.Step{{ID: 1, Status: enum.CIStatusPending}, {ID: 2, Status: enum.CIStatusError}}
	ds := ConvertToDroneSteps(steps)
	require.Len(t, ds, 2)
	require.Equal(t, "pending", ds[0].Status)

	back := ConvertFromDroneSteps(ds)
	require.Len(t, back, 2)
	require.Equal(t, enum.CIStatusError, back[1].Status)

	require.Equal(t, enum.CIStatusError, ConvertFromDroneStep(&drone.Step{Status: "garbage"}).Status)
	require.Equal(t, enum.CIStatusPending, ConvertFromDroneStep(&drone.Step{}).Status)
	require.Empty(t, ConvertToDroneSteps(nil))
	require.Empty(t, ConvertFromDroneSteps(nil))
}

func TestInfConvert_Line(t *testing.T) {
	line := ConvertFromDroneLine(&drone.Line{Number: 3, Message: "m", Timestamp: 4})
	require.Equal(t, 3, line.Number)
	require.Equal(t, "m", line.Message)
	require.EqualValues(t, 4, line.Timestamp)
}

func TestInfConvert_Build(t *testing.T) {
	e := &types.Execution{
		ID: 1, RepoID: 2, Trigger: "t", Number: 3, Parent: 4, Status: enum.CIStatusSuccess, Error: "e",
		Event: enum.TriggerEventPush, Action: enum.TriggerActionBranchUpdated, Link: "l", Timestamp: 5,
		Title: "ti", Message: "me", Before: "b", After: "a", Ref: "r", Fork: "f", Source: "s", Target: "tg",
		Author: "au", AuthorName: "an", AuthorEmail: "ae", AuthorAvatar: "av", Sender: "se",
		Params: map[string]string{"p": "v"}, Cron: "c", Deploy: "d", DeployID: 6, Debug: true,
		Started: 7000, Finished: 8000, Created: 9000, Updated: 10000, Version: 11,
	}
	b := ConvertToDroneBuild(e)
	require.EqualValues(t, 1, b.ID)
	require.Equal(t, "success", b.Status)
	require.Equal(t, "push", b.Event)
	require.Equal(t, "branch_updated", b.Action)
	require.EqualValues(t, 7, b.Started)
	require.EqualValues(t, 8, b.Finished)
	require.EqualValues(t, 9, b.Created)
	require.EqualValues(t, 10, b.Updated)
	require.True(t, b.Debug)
	require.Equal(t, map[string]string{"p": "v"}, b.Params)
}

func TestInfConvert_Repo(t *testing.T) {
	repo := &types.Repository{
		ID: 1, Identifier: "repo", CreatedBy: 2, Path: "space/repo", GitURL: "https://x.invalid/r.git",
		Created: 3, Updated: 4, Version: 5, DefaultBranch: "main",
	}
	private := ConvertToDroneRepo(repo, false)
	require.True(t, private.Private)
	require.True(t, private.Trusted)
	require.Equal(t, "space/repo", private.Namespace)
	require.Equal(t, "repo", private.Name)
	require.Equal(t, "main", private.Branch)
	require.EqualValues(t, 36000, private.Timeout)
	require.False(t, ConvertToDroneRepo(repo, true).Private)
}

func TestInfConvert_FileSecretsNetrc(t *testing.T) {
	require.Equal(t, "data", string(ConvertToDroneFile(&file.File{Data: []byte("data")}).Data))

	secrets := ConvertToDroneSecrets([]*types.Secret{{Identifier: "a", Data: "1"}, {Identifier: "b", Data: "2"}})
	require.Equal(t, "a", secrets[0].Name)
	require.Equal(t, "2", secrets[1].Data)
	require.Empty(t, ConvertToDroneSecrets(nil))

	require.Nil(t, ConvertToDroneNetrc(nil))
	netrc := ConvertToDroneNetrc(&Netrc{Machine: "m", Login: "l", Password: "p"})
	require.Equal(t, &drone.Netrc{Machine: "m", Login: "l", Password: "p"}, netrc)
}
