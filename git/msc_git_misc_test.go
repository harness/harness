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

package git

import (
	"testing"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/require"
)

func TestMscCreateEnvironmentForPush(t *testing.T) {
	ctx := WithRequestID(t.Context(), "msc-req")
	env := CreateEnvironmentForPush(ctx, WriteParams{
		RepoUID: "repo-1",
		Actor:   Identity{Name: "Actor", Email: "actor@test.invalid"},
		EnvVars: map[string]string{"CUSTOM": "1"},
	})
	require.Contains(t, env, EnvRequestID+"=msc-req")
	require.Contains(t, env, EnvRepoUID+"=repo-1")
	require.Contains(t, env, "CUSTOM=1")
}

func TestMscFileActionEnum(t *testing.T) {
	values := FileAction("").Enum()
	require.NotEmpty(t, values)
}

func TestMscOptimizeRepositoryParams(t *testing.T) {
	f := mscNewOpsFixture(t)
	require.NoError(t, OptimizeRepoStrategyGC.Validate())
	require.Error(t, OptimizeRepoStrategy(99).Validate())

	require.Error(t, f.svc.OptimizeRepository(t.Context(), OptimizeRepositoryParams{}))
	require.NoError(t, f.svc.OptimizeRepository(t.Context(), OptimizeRepositoryParams{
		ReadParams: f.mscRead(), Strategy: OptimizeRepoStrategyGC,
	}))
}

func TestMscDiffParamsValidate(t *testing.T) {
	require.Error(t, (&DiffParams{}).Validate())
	require.Error(t, (&DiffParams{ReadParams: ReadParams{RepoUID: "r"}, HeadRef: ""}).Validate())
}

func TestMscGetDiffHunkHeadersAndDiffCut(t *testing.T) {
	f := mscNewOpsFixture(t)
	ctx := t.Context()
	featureHead := mscCommitOnFeature(t, f)
	mainHead := sha.Must(f.mscHead(ctx, t))

	out, err := f.svc.GetDiffHunkHeaders(ctx, GetDiffHunkHeadersParams{
		ReadParams: f.mscRead(), SourceCommitSHA: featureHead.String(), TargetCommitSHA: mainHead.String(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, out.Files)

	_, err = f.svc.DiffCut(ctx, &DiffCutParams{
		ReadParams: f.mscRead(), SourceCommitSHA: mainHead.String(), TargetCommitSHA: mainHead.String(),
		Path: "a.txt",
	})
	require.True(t, errors.IsInvalidArgument(err))
}

func TestMscMergeVerifyValidate(t *testing.T) {
	f := mscNewOpsFixture(t)
	_, err := f.svc.MergeVerify(t.Context(), &MergeVerifyParams{ReadParams: f.mscRead()})
	require.Error(t, err)
}
