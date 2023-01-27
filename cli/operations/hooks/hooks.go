// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package hooks

import (
	"github.com/harness/gitness/internal/githook"

	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	// ParamHooks defines the parameter for the git hooks sub-commands.
	ParamHooks = "hooks"
)

func Register(app *kingpin.Application) {
	subCmd := app.Command(ParamHooks, "manage git server hooks")
	githook.RegisterAll(subCmd)
}
