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

package types

import (
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog"
)

type IDEArg string

const (
	VSCodeCustomizationArg IDEArg = "VSCODE_CUSTOMIZATION"
	VSCodeProxyURIArg      IDEArg = "VSCODE_PROXY_URI"
	IDERepoNameArg         IDEArg = "IDE_REPO_NAME"
	IDEDownloadURLArg      IDEArg = "IDE_DOWNLOAD_URL"
	IDEDIRNameArg          IDEArg = "IDE_DIR_NAME"
)

type GitspaceLogger interface {
	Info(msg string)
	Debug(msg string)
	Warn(msg string)
	Error(msg string, err error)
}

type ZerologAdapter struct {
	logger *zerolog.Logger
}

type DockerRegistryAuth struct {
	// only host name is required
	// eg: docker.io instead of https://docker.io
	RegistryURL string
	Username    *types.MaskSecret
	Password    *types.MaskSecret
}
