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

package agentlauncher

import (
	"context"
	"strconv"

	"github.com/harness/gitness/app/services/insights/triage"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// maxEscalationPaths bounds the paths passed to the CODEOWNERS suggestion.
const maxEscalationPaths = 100

// PathResolver returns the repository paths a task touches. It is best effort: the launcher only uses it to
// pick the human the task is handed to.
type PathResolver interface {
	TaskPaths(ctx context.Context, t *types.Task) ([]string, error)
}

// PRFiles returns the files changed by a pull request.
type PRFiles interface {
	ChangedFiles(ctx context.Context, pr *types.PullReq) ([]string, error)
}

// StorePathResolver resolves the paths of a task from its links: the in-app frames of the latest event of an
// Insights issue, the manifest of a Shield finding and the changed files of a pull request.
type StorePathResolver struct {
	Links interface {
		ListByTask(ctx context.Context, taskID int64) ([]*types.TaskLink, error)
	}
	Events interface {
		ListByIssue(ctx context.Context, issueID int64, limit int) ([]*types.InsightsEvent, error)
	}
	Findings interface {
		Find(ctx context.Context, id int64) (*types.ShieldFinding, error)
	}
	PullReqs interface {
		Find(ctx context.Context, id int64) (*types.PullReq, error)
	}
	PRFiles PRFiles
}

// TaskPaths implements PathResolver. A link that cannot be resolved is skipped.
func (r StorePathResolver) TaskPaths(ctx context.Context, t *types.Task) ([]string, error) {
	links, err := r.Links.ListByTask(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	var out []string
	seen := map[string]bool{}
	add := func(paths ...string) {
		for _, p := range paths {
			if p != "" && !seen[p] && len(out) < maxEscalationPaths {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	for _, l := range links {
		id, perr := strconv.ParseInt(l.RefID, 10, 64)
		if perr != nil {
			continue
		}
		switch l.Kind {
		case enum.TaskLinkKindInsightsIssue:
			evs, eerr := r.Events.ListByIssue(ctx, id, 1)
			if eerr != nil {
				log.Ctx(ctx).Debug().Err(eerr).Int64("issue_id", id).Msg("agent launcher: no events for escalation paths")
				continue
			}
			if len(evs) > 0 {
				add(triage.InAppPaths(evs[0])...)
			}
		case enum.TaskLinkKindShieldFinding:
			f, ferr := r.Findings.Find(ctx, id)
			if ferr != nil {
				log.Ctx(ctx).Debug().Err(ferr).Int64("finding_id", id).Msg("agent launcher: no finding for escalation paths")
				continue
			}
			add(f.ManifestPath)
		case enum.TaskLinkKindPullReq:
			if r.PullReqs == nil || r.PRFiles == nil {
				continue
			}
			pr, prerr := r.PullReqs.Find(ctx, id)
			if prerr != nil {
				continue
			}
			files, ferr := r.PRFiles.ChangedFiles(ctx, pr)
			if ferr != nil {
				log.Ctx(ctx).Debug().Err(ferr).Int64("pullreq_id", id).Msg("agent launcher: no pr files for escalation paths")
				continue
			}
			add(files...)
		case enum.TaskLinkKindAITask:
		}
	}
	return out, nil
}

// GitPRFiles lists the files changed by a pull request with the git service.
type GitPRFiles struct {
	Git   git.Interface
	Repos RepoFinder
}

// ChangedFiles implements PRFiles.
func (g GitPRFiles) ChangedFiles(ctx context.Context, pr *types.PullReq) ([]string, error) {
	repo, err := g.Repos.FindByID(ctx, pr.TargetRepoID)
	if err != nil {
		return nil, err
	}
	out, err := g.Git.DiffFileNames(ctx, &git.DiffParams{
		ReadParams: git.CreateReadParams(repo),
		BaseRef:    pr.MergeBaseSHA,
		HeadRef:    pr.SourceSHA,
	})
	if err != nil {
		return nil, err
	}
	return out.Files, nil
}
