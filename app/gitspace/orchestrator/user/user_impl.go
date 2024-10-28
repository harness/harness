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

package user

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/gitspace/orchestrator/common"
	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/template"
)

var _ Service = (*ServiceImpl)(nil)

const templateManagerUser = "manage_user.sh"

type ServiceImpl struct {
}

func NewUserServiceImpl() Service {
	return &ServiceImpl{}
}

func (u *ServiceImpl) Manage(ctx context.Context, exec *devcontainer.Exec) ([]byte, error) {
	osInfoScript := common.GetOSInfoScript()
	script, err := template.GenerateScriptFromTemplate(
		templateManagerUser, &template.SetupUserPayload{
			Username:     exec.UserIdentifier,
			AccessKey:    exec.AccessKey,
			AccessType:   exec.AccessType,
			HomeDir:      exec.HomeDir,
			OSInfoScript: osInfoScript,
		})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to generate scipt to manager user from template %s: %w", templateManagerUser, err)
	}

	output := "Setting up user inside container\n"
	_, err = exec.ExecuteCommandInHomeDirectory(ctx, script, true, false)
	if err != nil {
		return nil, fmt.Errorf("failed to setup user: %w", err)
	}

	output += "Successfully setup user\n"

	return []byte(output), nil
}
