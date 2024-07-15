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

//go:embed template/install_vscode_web.sh
var installScript string

const templateRunVSCodeWeb = "run_vscode_web.sh"

type VSCodeWebConfig struct {
	Port string
}

type VSCodeWeb struct {
	config *VSCodeWebConfig
}

func NewVsCodeWebService(config *VSCodeWebConfig) *VSCodeWeb {
	return &VSCodeWeb{config: config}
}

// Setup runs the installScript which downloads the required version of the code-server binary and runs it.
func (v *VSCodeWeb) Setup(ctx context.Context, devcontainer *Devcontainer, _ *types.GitspaceInstance) ([]byte, error) {
	output, err := devcontainer.ExecuteCommand(ctx, installScript, false)
	if err != nil {
		return nil, fmt.Errorf("failed to install code-server: %w", err)
	}

	runScript, err := GenerateScriptFromTemplate(
		templateRunVSCodeWeb, &RunVSCodeWebPayload{
			Port: v.config.Port,
		})
	if err != nil {
		return output, fmt.Errorf(
			"failed to generate scipt to run code-server from template %s: %w",
			templateRunVSCodeWeb,
			err,
		)
	}

	_, err = devcontainer.ExecuteCommand(ctx, runScript, true)
	if err != nil {
		return output, fmt.Errorf("failed to run code-server: %w", err)
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
