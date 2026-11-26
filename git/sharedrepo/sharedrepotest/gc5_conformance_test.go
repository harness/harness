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

package sharedrepotest_test

import (
	"testing"

	"github.com/harness/gitness/git/sharedrepo"
	"github.com/harness/gitness/git/sharedrepo/sharedrepotest"

	"github.com/stretchr/testify/require"
)

func gc5Factory(t *testing.T) sharedrepo.RunnerFactory {
	t.Helper()

	factory, err := sharedrepo.NewRunnerFactory(t.TempDir())
	require.NoError(t, err)

	return factory
}

func TestGc5RunnerConformance(t *testing.T) {
	sharedrepotest.RunnerConformance(t, gc5Factory(t))
}

func TestGc5ReadOnlyRunnerConformance(t *testing.T) {
	sharedrepotest.ReadOnlyRunnerConformance(t, gc5Factory(t))
}

func TestGc5RefRunnerConformance(t *testing.T) {
	sharedrepotest.RefRunnerConformance(t, gc5Factory(t))
}
