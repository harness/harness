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

package container

import (
	"context"
	"fmt"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	_ "embed"
)

var _ IDE = (*VSCodeWeb)(nil)

//go:embed template/run_vscode_web.sh
var runScript string

const templateInstallVSCodeWeb = "install_vscode_web.sh"

type VSCodeWebConfig struct {
	Port string
}

type VSCodeWeb struct {
	config *VSCodeWebConfig
}

func NewVsCodeWebService(config *VSCodeWebConfig) *VSCodeWeb {
	return &VSCodeWeb{config: config}
}

// Setup runs the installScript which downloads the required version of the code-server binary.
func (v *VSCodeWeb) Setup(ctx context.Context, devcontainer *Devcontainer, _ *types.GitspaceInstance) ([]byte, error) {
	installScript, err := GenerateScriptFromTemplate(
		templateInstallVSCodeWeb, &InstallVSCodeWebPayload{
			Port: v.config.Port,
		})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to generate scipt to install VSCode Web from template %s: %w",
			templateInstallVSCodeWeb,
			err,
		)
	}

	output, err := devcontainer.ExecuteCommand(ctx, installScript, false)
	if err != nil {
		return nil, fmt.Errorf("failed to install VSCode Web: %w", err)
	}

	return output, nil
}

// Run runs the code-server binary.
func (v *VSCodeWeb) Run(ctx context.Context, devcontainer *Devcontainer) ([]byte, error) {
	var output []byte

	_, err := devcontainer.ExecuteCommand(ctx, runScript, true)
	if err != nil {
		return nil, fmt.Errorf("failed to run VSCode Web: %w", err)
	}
	return output, nil
}

// PortAndProtocol returns the port on which the code-server is listening.
func (v *VSCodeWeb) PortAndProtocol() string {
	return v.config.Port + "/tcp"
}

func (v *VSCodeWeb) Type() enum.IDEType {
	return enum.IDETypeVSCodeWeb
}
