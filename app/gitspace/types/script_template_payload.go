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

import "github.com/harness/gitness/types/enum"

type CloneCodePayload struct {
	RepoURL  string
	Image    string
	Branch   string
	RepoName string
	Name     string
	Email    string
}

type SetupGitInstallPayload struct {
	OSInfoScript string
}

type SetupGitCredentialsPayload struct {
	CloneURLWithCreds string
}

type RunVSCodeWebPayload struct {
	Port      string
	Arguments string
	ProxyURI  string
}

type SetupVSCodeWebPayload struct {
	Extensions []string
}

type SetupUserPayload struct {
	Username   string
	AccessKey  string
	AccessType enum.GitspaceAccessType
	HomeDir    string
}

type SetupSSHServerPayload struct {
	Username     string
	AccessType   enum.GitspaceAccessType
	OSInfoScript string
}

type SetupVSCodeExtensionsPayload struct {
	Username   string
	Extensions string
	RepoName   string
}

type RunSSHServerPayload struct {
	Port string
}

type InstallToolsPayload struct {
	OSInfoScript string
}

type SupportedOSDistributionPayload struct {
	OSInfoScript string
}

type SetEnvPayload struct {
	EnvVariables []string
}

type SetupJetBrainsIDEPayload struct {
	Username            string
	IdeDownloadURLArm64 string
	IdeDownloadURLAmd64 string
	IdeDirName          string
}

type SetupJetBrainsPluginPayload struct {
	Username   string
	IdeDirName string
	// IdePluginsName contains like of plugins each separated with space.
	IdePluginsName string
}

type RunIntellijIDEPayload struct {
	Username       string
	RepoName       string
	IdeDownloadURL string
	IdeDirName     string
}
