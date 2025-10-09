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

package users

import (
	"context"
	"time"

	"github.com/harness/gitness/cli/provide"

	"gopkg.in/alecthomas/kingpin.v2"
)

type deleteCommand struct {
	email string
}

func (c *deleteCommand) run(*kingpin.ParseContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	return provide.Client().UserDelete(ctx, c.email)
}

// helper function registers the user delete command.
func registerDelete(app *kingpin.CmdClause) {
	c := &deleteCommand{}

	cmd := app.Command("delete", "delete a user").
		Action(c.run)

	cmd.Arg("id or email", "user id or email").
		Required().
		StringVar(&c.email)
}
