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

package account

import (
	"os"

	"github.com/harness/gitness/cli/provide"

	"gopkg.in/alecthomas/kingpin.v2"
)

type logoutCommand struct{}

func (c *logoutCommand) run(*kingpin.ParseContext) error {
	return os.Remove(provide.Session().Path())
}

// RegisterLogout helper function to register the logout command.
func RegisterLogout(app *kingpin.Application) {
	c := &logoutCommand{}

	app.Command("logout", "logout from the remote server").
		Action(c.run)
}
