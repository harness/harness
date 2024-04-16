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

package api

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/sha"

	"github.com/bmatcuk/doublestar/v4"
)

// FS represents a git filesystem.
type FS struct {
	ctx context.Context
	rev string
	dir string
}

// NewFS creates a new git filesystem.
func NewFS(ctx context.Context, rev, dir string) *FS {
	return &FS{
		ctx: ctx,
		rev: rev,
		dir: dir,
	}
}

func (f *FS) Open(path string) (fs.File, error) {
	treeNode, err := GetTreeNode(f.ctx, f.dir, f.rev, path, true)
	if err != nil {
		return nil, err
	}

	if treeNode.IsDir() {
		return nil, errors.InvalidArgument("can't open a directory")
	}
	if treeNode.IsLink() {
		return nil, errors.InvalidArgument("can't open a link")
	}
	if treeNode.IsSubmodule() {
		return nil, errors.InvalidArgument("can't open a submodule")
	}

	pipeRead, pipeWrite := io.Pipe()
	ctx, cancelFn := context.WithCancel(f.ctx)
	go func() {
		var err error

		defer func() {
			// If running of the command below fails, make the pipe reader also fail with the same error.
			_ = pipeWrite.CloseWithError(err)
		}()

		cmd := command.New("cat-file", command.WithFlag("-p"), command.WithArg(treeNode.SHA.String()))
		err = cmd.Run(f.ctx, command.WithDir(f.dir), command.WithStdout(pipeWrite))
	}()

	pathFile := &file{
		ctx:      ctx,
		cancelFn: cancelFn,
		path:     path,
		blobSHA:  treeNode.SHA,
		mode:     treeNode.Mode,
		size:     treeNode.Size,
		reader:   pipeRead,
	}

	return pathFile, nil
}

// ReadDir returns all entries for a directory.
// It is part of the fs.ReadDirFS interface.
func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	treeNodes, err := ListTreeNodes(f.ctx, f.dir, f.rev, name, true)
	if err != nil {
		return nil, fmt.Errorf("failed to read git directory: %w", err)
	}

	result := make([]fs.DirEntry, len(treeNodes))
	for i, treeNode := range treeNodes {
		result[i] = entry{treeNode}
	}

	return result, nil
}

// Glob returns all file names that match the pattern.
// It is part of the fs.GlobFS interface.
func (f *FS) Glob(pattern string) ([]string, error) {
	return doublestar.Glob(f, pattern)
}

// Stat returns entry file info for a file path.
// It is part of the fs.StatFS interface.
func (f *FS) Stat(name string) (fs.FileInfo, error) {
	treeInfo, err := GetTreeNode(f.ctx, f.dir, f.rev, name, true)
	if err != nil {
		return nil, fmt.Errorf("failed to read git directory: %w", err)
	}

	return entry{*treeInfo}, nil
}

type file struct {
	ctx      context.Context
	cancelFn context.CancelFunc

	path    string
	blobSHA sha.SHA
	mode    TreeNodeMode
	size    int64

	reader io.Reader
}

func (f *file) Stat() (fs.FileInfo, error) {
	return f, nil
}

// Read bytes from the file.
func (f *file) Read(bytes []byte) (int, error) {
	return f.reader.Read(bytes)
}

// Close closes the file.
func (f *file) Close() error {
	f.cancelFn()
	return nil
}

// Name returns the name of the file.
// It is part of the fs.FileInfo interface.
func (f *file) Name() string {
	return f.path
}

// Size returns file size - the size of the git blob object.
// It is part of the fs.FileInfo interface.
func (f *file) Size() int64 {
	return f.size
}

// Mode always returns 0 because a git blob is an ordinary file.
// It is part of the fs.FileInfo interface.
func (f *file) Mode() fs.FileMode {
	return 0
}

// ModTime is unimplemented.
// It is part of the fs.FileInfo interface.
func (f *file) ModTime() time.Time {
	// Git doesn't store file modification time directly.
	// It's possible to find the last commit (and thus the commit time)
	// that modified touched the file, but this is out of scope for this implementation.
	return time.Time{}
}

// IsDir implementation for the file struct always returns false.
// It is part of the fs.FileInfo interface.
func (f *file) IsDir() bool {
	return false
}

// Sys is unimplemented.
// It is part of the fs.FileInfo interface.
func (f *file) Sys() any {
	return nil
}

type entry struct {
	TreeNode
}

// Name returns name of a git tree entry.
// It is part of the fs.DirEntry interface.
func (e entry) Name() string {
	return e.TreeNode.Name
}

// IsDir returns if a git tree entry is a directory.
// It is part of the fs.FileInfo and fs.DirEntry interfaces.
func (e entry) IsDir() bool {
	return e.TreeNode.IsDir()
}

// Type returns the type of git tree entry.
// It is part of the fs.DirEntry interface.
func (e entry) Type() fs.FileMode {
	if e.TreeNode.IsDir() {
		return fs.ModeDir
	}
	return 0
}

// Info returns FileInfo for a git tree entry.
// It is part of the fs.DirEntry interface.
func (e entry) Info() (fs.FileInfo, error) {
	return e, nil
}

// Size returns file size - the size of the git blob object.
// It is part of the fs.FileInfo interface.
func (e entry) Size() int64 {
	return e.TreeNode.Size
}

// Mode always returns 0 because a git blob is an ordinary file.
// It is part of the fs.FileInfo interface.
func (e entry) Mode() fs.FileMode {
	return 0
}

// ModTime is unimplemented. Git doesn't store file modification time directly.
// It's possible to find the last commit (and thus the commit time)
// that modified touched the file, but this is out of scope for this implementation.
// It is part of the fs.FileInfo interface.
func (e entry) ModTime() time.Time { return time.Time{} }

// Sys is unimplemented.
// It is part of the fs.FileInfo interface.
func (e entry) Sys() any { return nil }
