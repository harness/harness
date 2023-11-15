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
	"io"
	"time"

	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
)

type SharedRepo interface {
	Path() string
	RemotePath() string
	Close(ctx context.Context)
	Clone(ctx context.Context, branchName string) error
	Init(ctx context.Context) error
	SetDefaultIndex(ctx context.Context) error
	LsFiles(ctx context.Context, filenames ...string) ([]string, error)
	RemoveFilesFromIndex(ctx context.Context, filenames ...string) error
	WriteGitObject(ctx context.Context, content io.Reader) (string, error)
	ShowFile(ctx context.Context, filePath, commitHash string, writer io.Writer) error
	AddObjectToIndex(ctx context.Context, mode, objectHash, objectPath string) error
	WriteTree(ctx context.Context) (string, error)
	GetLastCommit(ctx context.Context) (string, error)
	GetLastCommitByRef(ctx context.Context, ref string) (string, error)
	CommitTreeWithDate(
		ctx context.Context,
		parent string,
		author, committer *types.Identity,
		treeHash, message string,
		signoff bool,
		authorDate, committerDate time.Time,
	) (string, error)
	PushDeleteBranch(
		ctx context.Context,
		branch string,
		force bool,
		env ...string,
	) error
	PushCommitToBranch(
		ctx context.Context,
		commitSHA string,
		branch string,
		force bool,
		env ...string,
	) error
	PushBranch(
		ctx context.Context,
		sourceBranch string,
		branch string,
		force bool,
		env ...string,
	) error
	PushTag(
		ctx context.Context,
		tagName string,
		force bool,
		env ...string,
	) error
	PushDeleteTag(
		ctx context.Context,
		tagName string,
		force bool,
		env ...string,
	) error
	GetBranch(rev string) (*gitea.Branch, error)
	GetBranchCommit(branch string) (*gitea.Commit, error)
	GetCommit(commitID string) (*gitea.Commit, error)
}
