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

package migrate

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/store/database/migrate"

	"gopkg.in/alecthomas/kingpin.v2"
)

type commandCurrent struct {
	envfile string
}

func (c *commandCurrent) run(*kingpin.ParseContext) error {
	ctx := setupLoggingContext(context.Background())
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	db, err := getDB(ctx, c.envfile)

	if err != nil {
		return err
	}

	version, err := migrate.Current(ctx, db)
	if err != nil {
		return err
	}

	fmt.Println(version)

	return nil
}

func registerCurrent(app *kingpin.CmdClause) {
	c := &commandCurrent{}

	cmd := app.Command("current", "display the current version of the database").
		Action(c.run)

	cmd.Arg("envfile", "load the environment variable file").
		Default("").
		StringVar(&c.envfile)
}
