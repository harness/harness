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

package hook

import (
	"context"
	"errors"
	"os/signal"
	"syscall"

	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	// ParamPreReceive is the parameter under which the pre-receive operation is registered.
	ParamPreReceive = "pre-receive"
	// ParamUpdate is the parameter under which the update operation is registered.
	ParamUpdate = "update"
	// ParamPostReceive is the parameter under which the post-receive operation is registered.
	ParamPostReceive = "post-receive"

	// CommandNamePreReceive is the command used by git for the pre-receive hook
	// (os.args[0] == "hooks/pre-receive").
	CommandNamePreReceive = "hooks/pre-receive"
	// CommandNameUpdate is the command used by git for the update hook
	// (os.args[0] == "hooks/update").
	CommandNameUpdate = "hooks/update"
	// CommandNamePostReceive is the command used by git for the post-receive hook
	// (os.args[0] == "hooks/post-receive").
	CommandNamePostReceive = "hooks/post-receive"
)

// SanitizeArgsForGit sanitizes the command line arguments (os.Args) if the command indicates they are coming from git.
// Returns the santized args and true if the call comes from git, otherwise the original args are returned with false.
func SanitizeArgsForGit(command string, args []string) ([]string, bool) {
	switch command {
	case CommandNamePreReceive:
		return append([]string{ParamPreReceive}, args...), true
	case CommandNameUpdate:
		return append([]string{ParamUpdate}, args...), true
	case CommandNamePostReceive:
		return append([]string{ParamPostReceive}, args...), true
	default:
		return args, false
	}
}

// KingpinRegister is an abstraction of an entity that allows to register commands.
// This is required to allow registering hook commands both on application and sub command level.
type KingpinRegister interface {
	Command(name, help string) *kingpin.CmdClause
}

var (
	// ErrDisabled can be returned by the loading function to indicate the githook has been disabled.
	// Returning the error will cause the githook execution to be skipped (githook is noop and returns success).
	ErrDisabled = errors.New("githook disabled")
)

// LoadCLICoreFunc is a function that creates a new CLI core that's used for githook cli execution.
// This allows users to initialize their own CLI core with custom Client and configuration.
type LoadCLICoreFunc func() (*CLICore, error)

// RegisterAll registers all githook commands.
func RegisterAll(cmd KingpinRegister, loadCoreFn LoadCLICoreFunc) {
	RegisterPreReceive(cmd, loadCoreFn)
	RegisterUpdate(cmd, loadCoreFn)
	RegisterPostReceive(cmd, loadCoreFn)
}

// RegisterPreReceive registers the pre-receive githook command.
func RegisterPreReceive(cmd KingpinRegister, loadCoreFn LoadCLICoreFunc) {
	c := &preReceiveCommand{
		loadCoreFn: loadCoreFn,
	}

	cmd.Command(ParamPreReceive, "hook that is executed before any reference of the push is updated").
		Action(c.run)
}

// RegisterUpdate registers the update githook command.
func RegisterUpdate(cmd KingpinRegister, loadCoreFn LoadCLICoreFunc) {
	c := &updateCommand{
		loadCoreFn: loadCoreFn,
	}

	subCmd := cmd.Command(ParamUpdate, "hook that is executed before the specific reference gets updated").
		Action(c.run)

	subCmd.Arg("ref", "reference for which the hook is executed").
		Required().
		StringVar(&c.ref)

	subCmd.Arg("old", "old commit sha").
		Required().
		StringVar(&c.oldSHA)

	subCmd.Arg("new", "new commit sha").
		Required().
		StringVar(&c.newSHA)
}

// RegisterPostReceive registers the post-receive githook command.
func RegisterPostReceive(cmd KingpinRegister, loadCoreFn LoadCLICoreFunc) {
	c := &postReceiveCommand{
		loadCoreFn: loadCoreFn,
	}

	cmd.Command(ParamPostReceive, "hook that is executed after all references of the push got updated").
		Action(c.run)
}

type preReceiveCommand struct {
	loadCoreFn LoadCLICoreFunc
}

func (c *preReceiveCommand) run(*kingpin.ParseContext) error {
	return run(c.loadCoreFn, func(ctx context.Context, core *CLICore) error {
		return core.PreReceive(ctx)
	})
}

type updateCommand struct {
	loadCoreFn LoadCLICoreFunc

	ref    string
	oldSHA string
	newSHA string
}

func (c *updateCommand) run(*kingpin.ParseContext) error {
	return run(c.loadCoreFn, func(ctx context.Context, core *CLICore) error {
		return core.Update(ctx, c.ref, c.oldSHA, c.newSHA)
	})
}

type postReceiveCommand struct {
	loadCoreFn LoadCLICoreFunc
}

func (c *postReceiveCommand) run(*kingpin.ParseContext) error {
	return run(c.loadCoreFn, func(ctx context.Context, core *CLICore) error {
		return core.PostReceive(ctx)
	})
}

func run(loadCoreFn LoadCLICoreFunc, fn func(ctx context.Context, core *CLICore) error) error {
	core, err := loadCoreFn()
	if errors.Is(err, ErrDisabled) {
		// complete operation successfully without making any calls to the server.
		return nil
	}
	if err != nil {
		return err
	}

	// Create context that listens for the interrupt signal from the OS and has a timeout.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, core.executionTimeout)
	defer cancel()

	return fn(ctx, core)
}
