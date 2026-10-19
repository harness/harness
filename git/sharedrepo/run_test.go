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
	"testing"

	"github.com/harness/gitness/git/sharedrepo"
	"github.com/harness/gitness/git/sharedrepo/sharedrepotest"
)

func TestDefaultRunner(t *testing.T) {
	sharedrepotest.RunnerConformance(t, sharedrepo.DefaultRunner{})
}

func TestDefaultReadOnlyRunner(t *testing.T) {
	sharedrepotest.ReadOnlyRunnerConformance(t, sharedrepo.DefaultReadOnlyRunner{})
}
