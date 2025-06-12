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

package index

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/registry/app/utils/rpm"
)

const timeout = 3 * time.Minute

type Service interface {
	RegenerateRpmRepoData(ctx context.Context, registryID int64, rootParentID int64, rootIdentifier string) error
}

type service struct {
	rpmRegistryHelper rpm.RegistryHelper
	locker            *locker.Locker
}

func (s *service) RegenerateRpmRepoData(
	ctx context.Context,
	registryID int64,
	rootParentID int64,
	rootIdentifier string,
) error {
	unlock, err := s.locker.LockRpmRepoData(
		ctx,
		registryID,
		timeout,
	)
	if err != nil {
		return fmt.Errorf("failed to lock registry for RPM repo data regeneration: %w", err)
	}
	defer unlock()
	return s.rpmRegistryHelper.BuildRegistryFiles(ctx, registryID, rootParentID, rootIdentifier)
}

func NewService(
	rpmRegistryHelper rpm.RegistryHelper,
	locker *locker.Locker,
) Service {
	return &service{
		rpmRegistryHelper: rpmRegistryHelper,
		locker:            locker,
	}
}
