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

package command

import (
	"strings"
)

// Parse os args to Command object.
// This is very basic parser which doesn't care about
// flags or positional args values it just injects into proper
// slice of command struct. Every git command can contain
// globals:
//
//	git --help
//
// command:
//
//	git version
//	git diff
//
// action:
//
//	git remote set-url ...
//
// command or action flags:
//
//	git diff --shortstat
//
// command or action args:
//
//	git diff --shortstat main...dev
//
// post args:
//
//	git diff main...dev -- file1
func Parse(args ...string) *Command {
	actions := map[string]uint{}
	c := &Command{}

	globalPos := -1
	namePos := -1
	actionPos := -1
	flagsPos := -1
	argsPos := -1
	postPos := -1

	if len(args) == 0 {
		return c
	}

	if strings.ToLower(args[0]) == "git" {
		args = args[1:]
	}

	for i, arg := range args {
		isFlag := arg != "--" && strings.HasPrefix(arg, "-")
		b, isCommand := descriptions[arg]
		_, isAction := actions[arg]
		switch {
		case globalPos == -1 && namePos == -1 && isFlag:
			globalPos = i
		case namePos == -1 && isCommand:
			namePos = i
			actions = b.actions
		case actionPos == -1 && isAction && !isFlag:
			actionPos = i
		case flagsPos == -1 && (namePos >= 0 || actionPos > 0) && isFlag:
			flagsPos = i
		case argsPos == -1 && (namePos >= 0 || actionPos > 0) && !isFlag:
			argsPos = i
		case postPos == -1 && arg == "--":
			postPos = i
		}
	}

	end := len(args)

	if globalPos >= 0 {
		c.Globals = args[globalPos:cmpPos(namePos, end)]
	}

	if namePos >= 0 {
		c.Name = args[namePos]
	}

	if actionPos > 0 {
		c.Action = args[actionPos]
	}

	if flagsPos > 0 {
		c.Flags = args[flagsPos:cmpPos(argsPos, end)]
	}

	if argsPos > 0 {
		c.Args = args[argsPos:cmpPos(postPos, end)]
	}

	if postPos > 0 {
		c.PostSepArgs = args[postPos+1:]
	}

	return c
}

func cmpPos(check, or int) int {
	if check == -1 {
		return or
	}
	return check
}
