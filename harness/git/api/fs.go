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
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/sha"

	"github.com/bmatcuk/doublestar/v4"
)

// FS represents a git filesystem.
// It implements fs.FS interface.
type FS struct {
	ctx context.Context
	rev string
	dir string
}

var (
	// Make sure all targeted interfaces are implemented.
	_ fs.FS         = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
	_ fs.GlobFS     = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
)

// NewFS creates a new git file system for the provided revision.
func NewFS(ctx context.Context, rev, dir string) *FS {
	return &FS{
		ctx: ctx,
		rev: rev,
		dir: dir,
	}
}

// Open opens a file.
// It is part of the fs.FS interface.
func (f *FS) Open(path string) (fs.File, error) {
	if path != "" && !fs.ValidPath(path) {
		return nil, &fs.PathError{
			Op:   "open",
			Path: path,
			Err:  fs.ErrInvalid,
		}
	}

	treeNode, err := GetTreeNode(f.ctx, f.dir, f.rev, path, true)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, &fs.PathError{
				Op:   "open",
				Path: path,
				Err:  fs.ErrNotExist,
			}
		}

		return nil, &fs.PathError{
			Op:   "open",
			Path: path,
			Err:  err,
		}
	}

	switch {
	case treeNode.IsDir():
		return f.openTree(path, treeNode), nil
	case treeNode.IsSubmodule():
		return f.openSubmodule(path, treeNode), nil
	default:
		return f.openBlob(path, treeNode), nil
	}
}

func (f *FS) openBlob(path string, treeNode *TreeNode) *fsFile {
	pipeRead, pipeWrite := io.Pipe()
	ctx, cancelFn := context.WithCancel(f.ctx)
	go func() {
		cmd := command.New("cat-file", command.WithFlag("-p"), command.WithArg(treeNode.SHA.String()))
		_ = pipeWrite.CloseWithError(cmd.Run(ctx, command.WithDir(f.dir), command.WithStdout(pipeWrite)))
	}()
	go func() {
		<-ctx.Done()
		_ = pipeWrite.CloseWithError(ctx.Err())
	}()

	return &fsFile{
		ctx:      ctx,
		cancelFn: cancelFn,
		path:     path,
		blobSHA:  treeNode.SHA,
		mode:     treeNode.Mode,
		size:     treeNode.Size,
		reader:   pipeRead,
	}
}

func (f *FS) openSubmodule(path string, treeNode *TreeNode) *fsFile {
	content := treeNode.SHA.String() + "\n" // content of a submodule is the commit SHA plus end-of-line character.
	return &fsFile{
		ctx:      context.Background(),
		cancelFn: func() {},
		path:     path,
		blobSHA:  treeNode.SHA,
		mode:     treeNode.Mode,
		size:     int64(len(content)),
		reader:   strings.NewReader(content),
	}
}

func (f *FS) openTree(path string, treeNode *TreeNode) *fsDir {
	return &fsDir{
		ctx:     context.Background(),
		path:    path,
		treeSHA: treeNode.SHA,
		dir:     f.dir,
		skip:    0,
	}
}

// ReadFile reads the whole file.
// It is part of the fs.ReadFileFS interface.
func (f *FS) ReadFile(path string) ([]byte, error) {
	file, err := f.Open(path)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = file.Close()
	}()

	buffer := bytes.NewBuffer(nil)
	_, err = io.Copy(buffer, file)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// ReadDir returns all entries for a directory.
// It is part of the fs.ReadDirFS interface.
func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	treeNodes, err := ListTreeNodes(f.ctx, f.dir, f.rev, name, true, false)
	if err != nil {
		return nil, fmt.Errorf("failed to read git directory: %w", err)
	}

	result := make([]fs.DirEntry, len(treeNodes))
	for i, treeNode := range treeNodes {
		result[i] = fsEntry{treeNode}
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

	return fsEntry{*treeInfo}, nil
}

type fsDir struct {
	ctx context.Context

	path    string
	treeSHA sha.SHA

	dir  string
	skip int
}

// Stat returns fs.FileInfo for the directory.
// It is part of the fs.File interface.
func (d *fsDir) Stat() (fs.FileInfo, error) { return d, nil }

// Read always returns an error because it is not possible to read directory bytes.
// It is part of the fs.File interface.
func (d *fsDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{
		Op:   "read",
		Path: d.path,
		Err:  fs.ErrInvalid,
	}
}

// Close in a no-op for directories.
// It is part of the fs.File interface.
func (d *fsDir) Close() error { return nil }

// ReadDir lists entries in the directory. The integer parameter can be used for pagination.
// It is part of the fs.ReadDirFile interface.
func (d *fsDir) ReadDir(n int) ([]fs.DirEntry, error) {
	treeNodes, err := ListTreeNodes(d.ctx, d.dir, d.treeSHA.String(), "", true, false)
	if err != nil {
		return nil, fmt.Errorf("failed to read git directory: %w", err)
	}

	if d.skip >= len(treeNodes) {
		treeNodes = treeNodes[len(treeNodes):]
	} else {
		treeNodes = treeNodes[d.skip:]
	}

	var result []fs.DirEntry
	for _, treeNode := range treeNodes {
		result = append(result, fsEntry{treeNode})
		d.skip++

		if n >= 1 && n == len(result) {
			break
		}
	}

	if len(result) == 0 {
		return nil, io.EOF
	}

	return result, nil
}

// Name returns the path.
// It is part of the fs.FileInfo interface.
func (d *fsDir) Name() string { return d.path }

// Size implementation for directories returns zero.
// It is part of the fs.FileInfo interface.
func (d *fsDir) Size() int64 { return 0 }

// Mode always returns fs.ModeDir because a git tree is a directory.
// It is part of the fs.FileInfo interface.
func (d *fsDir) Mode() fs.FileMode { return fs.ModeDir }

// ModTime is unimplemented.
// It is part of the fs.FileInfo interface.
func (d *fsDir) ModTime() time.Time { return time.Time{} }

// IsDir implementation always returns true.
// It is part of the fs.FileInfo interface.
func (d *fsDir) IsDir() bool { return true }

// Sys is unimplemented.
// It is part of the fs.FileInfo interface.
func (d *fsDir) Sys() any { return nil }

type fsFile struct {
	ctx      context.Context
	cancelFn context.CancelFunc

	path    string
	blobSHA sha.SHA
	mode    TreeNodeMode
	size    int64

	reader io.Reader
}

// Stat returns fs.FileInfo for the file.
// It is part of the fs.File interface.
func (f *fsFile) Stat() (fs.FileInfo, error) { return f, nil }

// Read bytes from the file.
// It is part of the fs.File interface.
func (f *fsFile) Read(bytes []byte) (int, error) { return f.reader.Read(bytes) }

// Close closes the file.
// It is part of the fs.File interface.
func (f *fsFile) Close() error {
	f.cancelFn()
	return nil
}

// Name returns the name of the file.
// It is part of the fs.FileInfo interface.
func (f *fsFile) Name() string { return f.path }

// Size returns file size - the size of the git blob object.
// It is part of the fs.FileInfo interface.
func (f *fsFile) Size() int64 { return f.size }

// Mode returns file mode for the git blob.
// It is part of the fs.FileInfo interface.
func (f *fsFile) Mode() fs.FileMode {
	switch f.mode { //nolint:exhaustive
	case TreeNodeModeSymlink:
		return fs.ModeSymlink
	case TreeNodeModeCommit:
		return fs.ModeIrregular
	case TreeNodeModeExec:
		return 0o555
	default:
		return 0o444
	}
}

// ModTime is unimplemented. Git doesn't store file modification time directly.
// It's possible to find the last commit (and thus the commit time)
// that modified touched the file, but this is out of scope for this implementation.
// It is part of the fs.FileInfo interface.
func (f *fsFile) ModTime() time.Time { return time.Time{} }

// IsDir implementation always returns false.
// It is part of the fs.FileInfo interface.
func (f *fsFile) IsDir() bool { return false }

// Sys is unimplemented.
// It is part of the fs.FileInfo interface.
func (f *fsFile) Sys() any { return nil }

type fsEntry struct {
	TreeNode
}

// Name returns name of a git tree entry.
// It is part of the fs.DirEntry interface.
func (e fsEntry) Name() string { return e.TreeNode.Name }

// IsDir returns if a git tree entry is a directory.
// It is part of the fs.FileInfo and fs.DirEntry interfaces.
func (e fsEntry) IsDir() bool { return e.TreeNode.IsDir() }

// Type returns the type of git tree entry.
// It is part of the fs.DirEntry interface.
func (e fsEntry) Type() fs.FileMode { return e.Mode() }

// Info returns FileInfo for a git tree entry.
// It is part of the fs.DirEntry interface.
func (e fsEntry) Info() (fs.FileInfo, error) { return e, nil }

// Size returns file size - the size of the git blob object.
// It is part of the fs.FileInfo interface.
func (e fsEntry) Size() int64 { return e.TreeNode.Size }

// Mode always returns the filesystem entry mode.
// It is part of the fs.FileInfo interface.
func (e fsEntry) Mode() fs.FileMode {
	var mode fs.FileMode
	if e.TreeNode.Mode == TreeNodeModeExec {
		mode = 0o555
	} else {
		mode = 0o444
	}
	if e.TreeNode.IsDir() {
		mode |= fs.ModeDir
	}
	if e.TreeNode.IsLink() {
		mode |= fs.ModeSymlink
	}
	if e.TreeNode.IsSubmodule() {
		mode |= fs.ModeIrregular
	}
	return mode
}

// ModTime is unimplemented. Git doesn't store file modification time directly.
// It's possible to find the last commit (and thus the commit time)
// that modified touched the file, but this is out of scope for this implementation.
// It is part of the fs.FileInfo interface.
func (e fsEntry) ModTime() time.Time { return time.Time{} }

// Sys is unimplemented.
// It is part of the fs.FileInfo interface.
func (e fsEntry) Sys() any { return nil }
