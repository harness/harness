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

package driver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rsgWalkFake struct {
	StorageDriver
	list map[string][]string
	info map[string]FileInfoInternal
}

func (f *rsgWalkFake) List(_ context.Context, path string) ([]string, error) {
	return f.list[path], nil
}

func (f *rsgWalkFake) Stat(_ context.Context, path string) (FileInfo, error) {
	fi, ok := f.info[path]
	if !ok {
		return nil, PathNotFoundError{Path: path}
	}
	return fi, nil
}

func TestRsg_WalkFallbackVisitsFiles(t *testing.T) {
	ctx := context.Background()
	fake := &rsgWalkFake{
		list: map[string][]string{
			"/rsg":     {"/rsg/dir"},
			"/rsg/dir": {"/rsg/dir/file"},
		},
		info: map[string]FileInfoInternal{
			"/rsg/dir": {
				FileInfoFields: FileInfoFields{Path: "/rsg/dir", IsDir: true},
			},
			"/rsg/dir/file": {
				FileInfoFields: FileInfoFields{Path: "/rsg/dir/file", Size: 3, ModTime: time.Now()},
			},
		},
	}

	seen := []string{}
	err := WalkFallback(ctx, fake, "/rsg", func(fi FileInfo) error {
		if !fi.IsDir() {
			seen = append(seen, fi.Path())
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"/rsg/dir/file"}, seen)
}

func TestRsg_WalkFallbackErrFilledBuffer(t *testing.T) {
	ctx := context.Background()
	fake := &rsgWalkFake{
		list: map[string][]string{"/rsg": {"/rsg/a", "/rsg/b"}},
		info: map[string]FileInfoInternal{
			"/rsg/a": {FileInfoFields: FileInfoFields{Path: "/rsg/a", Size: 1}},
			"/rsg/b": {FileInfoFields: FileInfoFields{Path: "/rsg/b", Size: 1}},
		},
	}

	err := WalkFallback(ctx, fake, "/rsg", func(fi FileInfo) error {
		if fi.Path() == "/rsg/a" {
			return ErrFilledBuffer
		}
		return nil
	})
	require.NoError(t, err)
}

func TestRsg_WalkWithStartAfterHint(t *testing.T) {
	ctx := context.Background()
	fake := &rsgWalkFake{
		list: map[string][]string{
			"/rsg":     {"/rsg/a", "/rsg/b"},
			"/rsg/out": {},
		},
		info: map[string]FileInfoInternal{
			"/rsg/a": {FileInfoFields: FileInfoFields{Path: "/rsg/a", Size: 1}},
			"/rsg/b": {FileInfoFields: FileInfoFields{Path: "/rsg/b", Size: 1}},
		},
	}
	err := WalkFallback(ctx, fake, "/rsg", func(FileInfo) error { return nil },
		WithStartAfterHint("/rsg/a"))
	require.NoError(t, err)
}

func TestRsg_StorageDriverVersion(t *testing.T) {
	assert.Equal(t, uint(0), Version("0.1").Major())
	assert.Equal(t, uint(1), Version("0.1").Minor())
	assert.Equal(t, CurrentVersion, Version("0.1"))
}
