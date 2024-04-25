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

package repo

import (
	"context"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types/enum"
)

type GetCommitDivergencesInput struct {
	// MaxCount restricts the maximum number of diverging commits that are counted.
	// IMPORTANT: This restricts the total commit count, so a (5, 18) restricted to 10 will return (0, 10)
	MaxCount int32                     `json:"max_count"`
	Requests []CommitDivergenceRequest `json:"requests"`
}

// CommitDivergenceRequest contains the refs for which the converging commits should be counted.
type CommitDivergenceRequest struct {
	// From is the ref from which the counting of the diverging commits starts.
	From string `json:"from"`
	// To is the ref at which the counting of the diverging commits ends.
	// If the value is empty the divergence is calculated to the default branch of the repo.
	To string `json:"to"`
}

// CommitDivergence contains the information of the count of converging commits between two refs.
type CommitDivergence struct {
	// Ahead is the count of commits the 'From' ref is ahead of the 'To' ref.
	Ahead int32 `json:"ahead"`
	// Behind is the count of commits the 'From' ref is behind the 'To' ref.
	Behind int32 `json:"behind"`
}

// GetCommitDivergences returns the commit divergences between reference pairs.
func (c *Controller) GetCommitDivergences(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *GetCommitDivergencesInput,
) ([]CommitDivergence, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	// if no requests were provided return an empty list
	if in == nil || len(in.Requests) == 0 {
		return []CommitDivergence{}, nil
	}

	// if num of requests > page max return error
	if len(in.Requests) > request.PerPageMax {
		return nil, usererror.ErrRequestTooLarge
	}

	// map to rpc params
	options := &git.GetCommitDivergencesParams{
		ReadParams: git.CreateReadParams(repo),
		MaxCount:   in.MaxCount,
		Requests:   make([]git.CommitDivergenceRequest, len(in.Requests)),
	}
	for i := range in.Requests {
		options.Requests[i].From = in.Requests[i].From
		options.Requests[i].To = in.Requests[i].To
		// backfil default branch if no 'to' was provided
		if len(options.Requests[i].To) == 0 {
			options.Requests[i].To = repo.DefaultBranch
		}
	}

	// TODO: We should cache the responses as times can reach multiple seconds
	rpcOutput, err := c.git.GetCommitDivergences(ctx, options)
	if err != nil {
		return nil, err
	}

	// map to output type
	divergences := make([]CommitDivergence, len(rpcOutput.Divergences))
	for i := range rpcOutput.Divergences {
		divergences[i].Ahead = rpcOutput.Divergences[i].Ahead
		divergences[i].Behind = rpcOutput.Divergences[i].Behind
	}

	return divergences, nil
}
