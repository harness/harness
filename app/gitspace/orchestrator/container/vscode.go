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

	"github.com/harness/gitness/types/enum"
)

var _ IDE = (*VSCode)(nil)

type VSCode struct{}

func NewVsCodeService() *VSCode {
	return &VSCode{}
}

// Setup is a NOOP since VS Code doesn't require any installation.
// TODO Check if the SSH server is accessible on the required port.
func (v *VSCode) Setup(_ context.Context, _ *Devcontainer) ([]byte, error) {
	return nil, nil
}

// PortAndProtocol return nil since VS Code doesn't require any additional port to be exposed.
func (v *VSCode) PortAndProtocol() string {
	return ""
}

func (v *VSCode) Type() enum.IDEType {
	return enum.IDETypeVSCode
}
