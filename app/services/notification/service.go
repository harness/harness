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

package notification

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
)

const (
	eventReaderGroupName = "gitness:notification"
	templatesDir         = "templates"
	subjectPullReqEvent  = "[%s] %s (PR #%d)"
)

var (
	//go:embed  templates/*
	files         embed.FS
	htmlTemplates map[string]*template.Template
)

func init() {
	err := LoadTemplates()
	if err != nil {
		panic(err)
	}
}

func LoadTemplates() error {
	htmlTemplates = make(map[string]*template.Template)
	tmplFiles, err := fs.ReadDir(files, templatesDir)
	if err != nil {
		return err
	}

	for _, tmpl := range tmplFiles {
		if tmpl.IsDir() {
			continue
		}

		pt, err := template.ParseFS(files, path.Join(templatesDir, tmpl.Name()))
		if err != nil {
			return err
		}

		htmlTemplates[tmpl.Name()] = pt
	}
	return nil
}

type BasePullReqPayload struct {
	Repo       *types.Repository
	PullReq    *types.PullReq
	Author     *types.PrincipalInfo
	PullReqURL string
}

type Config struct {
	EventReaderName string
	Concurrency     int
	MaxRetries      int
}

type Service struct {
	config                Config
	notificationClient    Client
	prReaderFactory       *events.ReaderFactory[*pullreqevents.Reader]
	pullReqStore          store.PullReqStore
	repoStore             store.RepoStore
	principalInfoView     store.PrincipalInfoView
	principalInfoCache    store.PrincipalInfoCache
	pullReqReviewersStore store.PullReqReviewerStore
	pullReqActivityStore  store.PullReqActivityStore
	spacePathStore        store.SpacePathStore
	urlProvider           url.Provider
}

func NewService(
	ctx context.Context,
	config Config,
	notificationClient Client,
	prReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
	pullReqStore store.PullReqStore,
	repoStore store.RepoStore,
	principalInfoView store.PrincipalInfoView,
	principalInfoCache store.PrincipalInfoCache,
	pullReqReviewersStore store.PullReqReviewerStore,
	pullReqActivityStore store.PullReqActivityStore,
	spacePathStore store.SpacePathStore,
	urlProvider url.Provider,
) (*Service, error) {
	service := &Service{
		config:                config,
		notificationClient:    notificationClient,
		prReaderFactory:       prReaderFactory,
		pullReqStore:          pullReqStore,
		repoStore:             repoStore,
		principalInfoView:     principalInfoView,
		principalInfoCache:    principalInfoCache,
		pullReqReviewersStore: pullReqReviewersStore,
		pullReqActivityStore:  pullReqActivityStore,
		spacePathStore:        spacePathStore,
		urlProvider:           urlProvider,
	}

	_, err := service.prReaderFactory.Launch(
		ctx,
		eventReaderGroupName,
		config.EventReaderName,
		func(r *pullreqevents.Reader,
		) error {
			r.Configure(
				stream.WithConcurrency(config.Concurrency),
				stream.WithHandlerOptions(
					stream.WithMaxRetries(config.MaxRetries),
				))

			_ = r.RegisterReviewerAdded(service.notifyReviewerAdded)
			_ = r.RegisterCommentCreated(service.notifyCommentCreated)
			_ = r.RegisterBranchUpdated(service.notifyPullReqBranchUpdated)
			_ = r.RegisterReviewSubmitted(service.notifyReviewSubmitted)

			// state changes
			_ = r.RegisterMerged(service.notifyPullReqStateMerged)
			_ = r.RegisterClosed(service.notifyPullReqStateClosed)
			_ = r.RegisterReopened(service.notifyPullReqStateReOpened)
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch event reader for %s: %w", eventReaderGroupName, err)
	}

	return service, nil
}

func (s *Service) getBasePayload(
	ctx context.Context,
	base pullreqevents.Base,
) (*BasePullReqPayload, error) {
	repo, err := s.repoStore.Find(ctx, base.TargetRepoID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repo from repoStore: %w", err)
	}
	pullReq, err := s.pullReqStore.Find(ctx, base.PullReqID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pullreq from pullReqStore: %w", err)
	}

	author, err := s.principalInfoCache.Get(ctx, pullReq.CreatedBy)
	if err != nil {
		return nil,
			fmt.Errorf("failed to fetch author %d from principalInfoCache while building base notification: %w",
				pullReq.CreatedBy, err)
	}

	return &BasePullReqPayload{
		Repo:       repo,
		PullReq:    pullReq,
		Author:     author,
		PullReqURL: s.urlProvider.GenerateUIPRURL(ctx, repo.Path, pullReq.Number),
	}, nil
}
