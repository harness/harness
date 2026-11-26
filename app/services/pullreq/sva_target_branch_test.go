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

package pullreq

import (
	"context"
	"testing"

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

// svaRefGit resolves the default branch, computes the merge base and the diff statistics.
type svaRefGit struct {
	git.Interface

	getRefErr    error
	mergeBaseErr error
	mergeBaseSHA string
	diffStatsErr error
}

func (g *svaRefGit) GetRef(context.Context, git.GetRefParams) (git.GetRefResponse, error) {
	return git.GetRefResponse{SHA: svaSHA("a")}, g.getRefErr
}

func (g *svaRefGit) MergeBase(context.Context, git.MergeBaseParams) (git.MergeBaseOutput, error) {
	return git.MergeBaseOutput{MergeBaseSHA: svaSHA(g.mergeBaseSHA)}, g.mergeBaseErr
}

func (g *svaRefGit) DiffStats(context.Context, *git.DiffParams) (git.DiffStatsOutput, error) {
	return git.DiffStatsOutput{}, g.diffStatsErr
}

func TestSvaUpdatePRToDefaultBranchFailures(t *testing.T) {
	t.Parallel()

	repo := &types.Repository{ID: 1, DefaultBranch: "main"}
	sourceSHA := svaSHA("c").String()

	newPR := func(state enum.PullReqState) *types.PullReq {
		return &types.PullReq{
			ID: 9, Number: 4, State: state, SourceSHA: sourceSHA, TargetBranch: "gone", MergeBaseSHA: "oldbase",
		}
	}

	tests := []struct {
		name    string
		git     *svaRefGit
		state   enum.PullReqState
		update  func(context.Context, *types.PullReq, func(*types.PullReq) error) (*types.PullReq, error)
		wantErr bool
	}{
		{
			name:    "default branch can't be resolved",
			git:     &svaRefGit{getRefErr: errSvaBoom, mergeBaseSHA: "b"},
			state:   enum.PullReqStateOpen,
			wantErr: true,
		},
		{
			name:    "merge base failure",
			git:     &svaRefGit{mergeBaseErr: errSvaBoom, mergeBaseSHA: "b"},
			state:   enum.PullReqStateOpen,
			wantErr: true,
		},
		{
			name:    "diff stats failure",
			git:     &svaRefGit{diffStatsErr: errSvaBoom, mergeBaseSHA: "b"},
			state:   enum.PullReqStateOpen,
			wantErr: true,
		},
		{
			name:  "pull request got merged in the meantime",
			git:   &svaRefGit{mergeBaseSHA: "b"},
			state: enum.PullReqStateMerged,
		},
		{
			name:  "update failure",
			git:   &svaRefGit{mergeBaseSHA: "b"},
			state: enum.PullReqStateOpen,
			update: func(context.Context, *types.PullReq, func(*types.PullReq) error) (*types.PullReq, error) {
				return nil, errSvaBoom
			},
			wantErr: true,
		},
		{
			name:  "no new commits and the pull request is no longer open",
			git:   &svaRefGit{mergeBaseSHA: "c"},
			state: enum.PullReqStateMerged,
		},
		{
			name:  "no new commits and the pull request can't be closed",
			git:   &svaRefGit{mergeBaseSHA: "c"},
			state: enum.PullReqStateOpen,
			update: func(context.Context, *types.PullReq, func(*types.PullReq) error) (*types.PullReq, error) {
				return nil, errSvaBoom
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			update := test.update
			if update == nil {
				update = svaApplyUpdate
			}

			svc := &Service{
				git:          test.git,
				pullreqStore: &svaPullReqStore{updateOptLck: update},
			}

			err := svc.updatePRToDefaultBranch(context.Background(), newPR(test.state), repo, "gone", 5)
			if test.wantErr {
				require.ErrorIs(t, err, errSvaBoom)
				return
			}
			require.NoError(t, err)
		})
	}
}
