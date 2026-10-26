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

package sharedrepo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/harness/gitness/git/sharedrepo"
	"github.com/harness/gitness/git/sharedrepo/sharedrepotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRunnerFactory(t *testing.T) sharedrepo.RunnerFactory {
	t.Helper()

	factory, err := sharedrepo.NewRunnerFactory(t.TempDir())
	require.NoError(t, err)

	return factory
}

func TestDefaultRunner(t *testing.T) {
	sharedrepotest.RunnerConformance(t, newTestRunnerFactory(t))
}

func TestDefaultReadOnlyRunner(t *testing.T) {
	sharedrepotest.ReadOnlyRunnerConformance(t, newTestRunnerFactory(t))
}

func TestDefaultRefRunner(t *testing.T) {
	sharedrepotest.RefRunnerConformance(t, newTestRunnerFactory(t))
}

func TestNewRunnerFactory(t *testing.T) {
	root := t.TempDir()

	factory, err := sharedrepo.NewRunnerFactory(root)
	require.NoError(t, err)

	dir := factory.Directory()
	assert.Equal(t, filepath.Join(root, "shared_temp"), dir)
	assert.DirExists(t, dir)
}

func TestNewRunnerFactory_FailsIfDirectoryCannotBeCreated(t *testing.T) {
	// a regular file in place of the root directory prevents creation of its subdirectory
	root := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(root, nil, 0o600))

	_, err := sharedrepo.NewRunnerFactory(root)
	require.Error(t, err)
}
