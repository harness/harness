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
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types/enum"
)

type InfoRefsParams struct {
	ReadParams
	Service     string
	Options     []string // (key, value) pair
	GitProtocol string
}

func (s *Service) GetInfoRefs(ctx context.Context, w io.Writer, params *InfoRefsParams) error {
	if err := params.Validate(); err != nil {
		return err
	}

	environ := []string{}
	if params.GitProtocol != "" {
		environ = append(environ, "GIT_PROTOCOL="+params.GitProtocol)
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)
	err := s.git.InfoRefs(ctx, repoPath, params.Service, w, environ...)
	if err != nil {
		return fmt.Errorf("failed to fetch info references: %w", err)
	}
	return nil
}

type ServicePackParams struct {
	*ReadParams
	*WriteParams
	api.ServicePackOptions
}

func (p *ServicePackParams) Validate() error {
	if p.Service == "" {
		return errors.InvalidArgument("service is mandatory field")
	}
	return nil
}

func (s *Service) ServicePack(ctx context.Context, params *ServicePackParams) error {
	if err := params.Validate(); err != nil {
		return err
	}
	var repoPath string
	switch params.Service {
	case enum.GitServiceTypeUploadPack:
		if err := params.ReadParams.Validate(); err != nil {
			return errors.InvalidArgument("upload-pack requires ReadParams")
		}
		repoPath = getFullPathForRepo(s.reposRoot, params.ReadParams.RepoUID)
	case enum.GitServiceTypeReceivePack:
		if err := params.WriteParams.Validate(); err != nil {
			return errors.InvalidArgument("receive-pack requires WriteParams")
		}
		params.Env = append(params.Env, CreateEnvironmentForPush(ctx, *params.WriteParams)...)
		repoPath = getFullPathForRepo(s.reposRoot, params.WriteParams.RepoUID)
	default:
		return errors.InvalidArgument("unsupported service provided: %s", params.Service)
	}

	err := s.git.ServicePack(ctx, repoPath, params.ServicePackOptions)
	if err != nil {
		return fmt.Errorf("failed to execute git %s: %w", params.Service, err)
	}

	return nil
}
