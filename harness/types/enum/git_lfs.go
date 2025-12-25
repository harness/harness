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

package enum

import (
	"fmt"
	"strings"
)

type GitLFSTransferType string

const (
	GitLFSTransferTypeBasic GitLFSTransferType = "basic"
	GitLFSTransferTypeSSH   GitLFSTransferType = "ssh"
	// TODO GitLFSTransferTypeMultipart
)

func ParseGitLFSTransferType(s string) (GitLFSTransferType, error) {
	switch strings.ToLower(s) {
	case string(GitLFSTransferTypeBasic):
		return GitLFSTransferTypeBasic, nil
	case string(GitLFSTransferTypeSSH):
		return GitLFSTransferTypeSSH, nil
	default:
		return "", fmt.Errorf("unknown git-lfs transfer type provided: %q", s)
	}
}

type GitLFSOperationType string

const (
	GitLFSOperationTypeDownload GitLFSOperationType = "download"
	GitLFSOperationTypeUpload   GitLFSOperationType = "upload"
)

func ParseGitLFSOperationType(s string) (GitLFSOperationType, error) {
	switch strings.ToLower(s) {
	case string(GitLFSOperationTypeDownload):
		return GitLFSOperationTypeDownload, nil
	case string(GitLFSOperationTypeUpload):
		return GitLFSOperationTypeUpload, nil
	default:
		return "", fmt.Errorf("unknown git-lfs operation type provided: %q", s)
	}
}

// GitLFSServiceType represents the different types of services git-lfs client sends over ssh.
type GitLFSServiceType string

const (
	// GitLFSServiceTypeTransfer is sent by git lfs client for transfer LFS objects.
	GitLFSServiceTypeTransfer GitLFSServiceType = "git-lfs-transfer"
	// GitLFSServiceTypeAuthenticate is sent by git lfs client for authentication.
	GitLFSServiceTypeAuthenticate GitLFSServiceType = "git-lfs-authenticate"
)

func ParseGitLFSServiceType(s string) (GitLFSServiceType, error) {
	switch strings.ToLower(s) {
	case string(GitLFSServiceTypeTransfer):
		return GitLFSServiceTypeTransfer, nil
	case string(GitLFSServiceTypeAuthenticate):
		return GitLFSServiceTypeAuthenticate, nil
	default:
		return "", fmt.Errorf("unknown git-lfs service type provided: %q", s)
	}
}
