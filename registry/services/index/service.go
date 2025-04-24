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

	"github.com/harness/gitness/registry/app/utils/rpm"
)

type Service interface {
	RegenerateRpmRepoData(ctx context.Context, registryID int64, rootParentID int64, rootIdentifier string) error
}

type service struct {
	rpmRegistryHelper rpm.RegistryHelper
}

func (s *service) RegenerateRpmRepoData(
	ctx context.Context,
	registryID int64,
	rootParentID int64,
	rootIdentifier string,
) error {
	// TODO: integrate with distributed lock
	return s.rpmRegistryHelper.BuildRegistryFiles(ctx, registryID, rootParentID, rootIdentifier)
}

func NewService(
	rpmRegistryHelper rpm.RegistryHelper,
) Service {
	return &service{
		rpmRegistryHelper: rpmRegistryHelper,
	}
}
