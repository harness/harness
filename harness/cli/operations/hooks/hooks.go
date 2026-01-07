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

package hooks

import (
	gitnessgithook "github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/git/hook"

	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	// ParamHooks defines the parameter for the git hooks sub-commands.
	ParamHooks = "hooks"
)

func Register(app *kingpin.Application) {
	subCmd := app.Command(ParamHooks, "manage git server hooks")
	hook.RegisterAll(subCmd, gitnessgithook.LoadFromEnvironment)
}
