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

package cli

import (
	"os"
	"testing"

	"github.com/harness/gitness/cli/operations/hooks"
	"github.com/harness/gitness/git/hook"

	"github.com/stretchr/testify/require"
)

func TestMscGetArguments(t *testing.T) {
	old := os.Args
	t.Cleanup(func() { os.Args = old })

	os.Args = []string{"gitness", "users", "list"}
	require.Equal(t, []string{"users", "list"}, GetArguments())

	os.Args = []string{hook.CommandNamePreReceive}
	require.Equal(t, []string{hooks.ParamHooks, hook.ParamPreReceive}, GetArguments())

	os.Args = []string{hook.CommandNameUpdate, "refs/heads/main", "old", "new"}
	require.Equal(t, []string{hooks.ParamHooks, hook.ParamUpdate, "refs/heads/main", "old", "new"}, GetArguments())
}
