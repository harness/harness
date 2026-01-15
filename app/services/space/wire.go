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

package space

import (
	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/services/gitspace"
	"github.com/harness/gitness/app/services/infraprovider"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideService,
)

func ProvideService(
	tx dbtx.Transactor,
	scheduler *job.Scheduler,
	executor *job.Executor,
	encrypter encrypt.Encrypter,
	repoStore store.RepoStore,
	spaceStore store.SpaceStore,
	spacePathStore store.SpacePathStore,
	rulesStore store.RuleStore,
	// resourceMover is nil for gitness standalone
	resourceMover ResourceMover,
	spaceFinder refcache.SpaceFinder,
	gitspaceSvs *gitspace.Service,
	infraProviderSvc *infraprovider.Service,
	repoCtrl *repo.Controller,
) (*Service, error) {
	service := NewService(
		tx,
		scheduler,
		encrypter,
		repoStore,
		spaceStore,
		spacePathStore,
		rulesStore,
		resourceMover,
		spaceFinder,
		gitspaceSvs,
		infraProviderSvc,
		repoCtrl,
	)
	err := executor.Register(jobType, service)
	if err != nil {
		return nil, err
	}

	return service, nil
}
