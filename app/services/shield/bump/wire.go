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

package bump

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/controller/pullreq"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/services/agentguard"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/services/shield"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideService,
)

// ProvideService provides the bump service, registers its job handler and hooks it to the scans.
func ProvideService(
	config *types.Config,
	guard *agentguard.Guard,
	settingsService *settings.Service,
	gitSvc git.Interface,
	pullreqCtrl *pullreq.Controller,
	pullreqStore store.PullReqStore,
	urlProvider url.Provider,
	taskSvc *task.Service,
	taskStore store.TaskStore,
	findings store.ShieldFindingStore,
	repos refcache.RepoFinder,
	scheduler *job.Scheduler,
	executor *job.Executor,
	shieldSvc *shield.Service,
) (*Service, error) {
	svc := NewService(
		Config{
			Enabled:        config.Shield.Enabled,
			PatchThreshold: config.Shield.PatchThreshold,
			GoProxy:        config.Shield.GoProxy,
		},
		guard,
		agentguard.ServiceSettings{Service: settingsService},
		&gitRepository{git: gitSvc, urls: urlProvider},
		&pullRequests{ctrl: pullreqCtrl, store: pullreqStore},
		taskAdapter{svc: taskSvc, store: taskStore},
		findings,
		repos,
		scheduler,
		bootstrap.GetAgentServicePrincipal,
	)
	if err := executor.Register(JobTypeBump, svc.Handler()); err != nil {
		return nil, fmt.Errorf("failed to register shield bump job: %w", err)
	}
	shieldSvc.SetFindingsHook(svc.OnFindings)
	return svc, nil
}

func agentSession() (*auth.Session, error) {
	p := bootstrap.GetAgentServicePrincipal()
	if p == nil {
		return nil, errors.Internal(nil, "agent service principal is not set up")
	}
	return &auth.Session{Principal: *p, Metadata: &auth.EmptyMetadata{}}, nil
}

type taskAdapter struct {
	svc   *task.Service
	store store.TaskStore
}

func (a taskAdapter) Find(ctx context.Context, id int64) (*types.Task, error) {
	return a.store.Find(ctx, id)
}

func (a taskAdapter) Link(ctx context.Context, principalID, taskID int64, kind enum.TaskLinkKind, refID string) error {
	return a.svc.Link(ctx, principalID, taskID, kind, refID)
}

func (a taskAdapter) AddComment(ctx context.Context, principalID, taskID int64, text string) error {
	return a.svc.AddComment(ctx, principalID, taskID, text)
}

// gitRepository implements Repository on top of the git service.
type gitRepository struct {
	git  git.Interface
	urls url.Provider
}

func (g *gitRepository) HeadSHA(ctx context.Context, repo *types.RepositoryCore, branch string) (string, error) {
	out, err := g.git.GetBranch(ctx, &git.GetBranchParams{
		ReadParams: git.CreateReadParams(repo), BranchName: branch,
	})
	if err != nil {
		return "", err
	}
	return out.Branch.SHA.String(), nil
}

func (g *gitRepository) ReadFile(
	ctx context.Context,
	repo *types.RepositoryCore,
	ref, filePath string,
) (File, bool, error) {
	read := git.CreateReadParams(repo)
	node, err := g.git.GetTreeNode(ctx, &git.GetTreeNodeParams{ReadParams: read, GitREF: ref, Path: filePath})
	if errors.IsNotFound(err) {
		return File{}, false, nil
	}
	if err != nil {
		return File{}, false, err
	}
	const limit = 32 << 20
	blob, err := g.git.GetBlob(ctx, &git.GetBlobParams{ReadParams: read, SHA: node.Node.SHA, SizeLimit: limit})
	if err != nil {
		return File{}, false, err
	}
	defer blob.Content.Close()
	if blob.Size > limit {
		return File{}, false, fmt.Errorf("%s is too large", filePath)
	}
	data, err := io.ReadAll(blob.Content)
	if err != nil {
		return File{}, false, err
	}
	return File{Content: data, BlobSHA: node.Node.SHA}, true, nil
}

func (g *gitRepository) BranchExists(ctx context.Context, repo *types.RepositoryCore, branch string) (bool, error) {
	_, err := g.git.GetBranch(ctx, &git.GetBranchParams{ReadParams: git.CreateReadParams(repo), BranchName: branch})
	if errors.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

func (g *gitRepository) CreateBranch(ctx context.Context, repo *types.RepositoryCore, branch, fromSHA string) error {
	session, err := agentSession()
	if err != nil {
		return err
	}
	wp, err := controller.CreateRPCAPIRefsWriteParams(ctx, g.urls, session, repo)
	if err != nil {
		return err
	}
	_, err = g.git.CreateBranch(ctx, &git.CreateBranchParams{WriteParams: wp, BranchName: branch, Target: fromSHA})
	return err
}

func (g *gitRepository) Commit(
	ctx context.Context,
	repo *types.RepositoryCore,
	branch, message string,
	files []FileChange,
) error {
	session, err := agentSession()
	if err != nil {
		return err
	}
	wp, err := controller.CreateRPCAPIContentWriteParams(ctx, g.urls, session, repo)
	if err != nil {
		return err
	}
	actions := make([]git.CommitFileAction, 0, len(files))
	for _, f := range files {
		a := git.CommitFileAction{Action: git.CreateAction, Path: f.Path, Payload: f.Content}
		if f.BlobSHA != "" {
			a.Action = git.UpdateAction
			a.SHA = sha.Must(f.BlobSHA)
		}
		actions = append(actions, a)
	}
	id := controller.IdentityFromPrincipalInfo(*session.Principal.ToPrincipalInfo())
	_, err = g.git.CommitFiles(ctx, &git.CommitFilesParams{
		WriteParams: wp,
		Message:     message,
		Branch:      branch,
		Committer:   id,
		Author:      id,
		Actions:     actions,
	})
	return err
}

// pullRequests implements PullRequests on top of the pull request controller.
type pullRequests struct {
	ctrl  *pullreq.Controller
	store store.PullReqStore
}

func (p *pullRequests) HasOpen(
	ctx context.Context,
	repo *types.RepositoryCore,
	sourceBranch, targetBranch string,
) (bool, error) {
	existing, err := p.store.List(ctx, &types.PullReqFilter{
		SourceRepoID:       repo.ID,
		SourceBranch:       sourceBranch,
		TargetRepoID:       repo.ID,
		TargetBranch:       targetBranch,
		States:             []enum.PullReqState{enum.PullReqStateOpen},
		Size:               1,
		Sort:               enum.PullReqSortNumber,
		Order:              enum.OrderAsc,
		ExcludeDescription: true,
	})
	if err != nil {
		return false, err
	}
	return len(existing) > 0, nil
}

func (p *pullRequests) Create(
	ctx context.Context,
	repo *types.RepositoryCore,
	in PullRequestInput,
) (PullRequest, error) {
	session, err := agentSession()
	if err != nil {
		return PullRequest{}, err
	}
	pr, err := p.ctrl.Create(ctx, session, repo.Path, &pullreq.CreateInput{
		IsDraft:      in.IsDraft,
		Title:        in.Title,
		Description:  in.Description,
		SourceBranch: in.SourceBranch,
		TargetBranch: in.TargetBranch,
	})
	if err != nil {
		return PullRequest{}, err
	}
	return PullRequest{ID: pr.ID, Number: pr.Number}, nil
}
