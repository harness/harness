// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package githook

import (
	"context"
	"errors"
	"os/signal"
	"syscall"
	"time"

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

var (
	// ExecutionTimeout is the timeout used for githook CLI runs.
	ExecutionTimeout = 3 * time.Minute
)

// SanitizeArgsForGit sanitizes the command line arguments (os.Args) if the command indicates they are comming from git.
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

// RegisterAll registers all githook commands.
func RegisterAll(cmd KingpinRegister) {
	RegisterPreReceive(cmd)
	RegisterUpdate(cmd)
	RegisterPostReceive(cmd)
}

// RegisterPreReceive registers the pre-receive githook command.
func RegisterPreReceive(cmd KingpinRegister) {
	c := &preReceiveCommand{}

	cmd.Command(ParamPreReceive, "hook that is executed before any reference of the push is updated").
		Action(c.run)
}

// RegisterUpdate registers the update githook command.
func RegisterUpdate(cmd KingpinRegister) {
	c := &updateCommand{}

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
func RegisterPostReceive(cmd KingpinRegister) {
	c := &postReceiveCommand{}

	cmd.Command(ParamPostReceive, "hook that is executed after all references of the push got updated").
		Action(c.run)
}

type preReceiveCommand struct{}

func (c *preReceiveCommand) run(*kingpin.ParseContext) error {
	return run(func(ctx context.Context, hook *GitHook) error {
		return hook.PreReceive(ctx)
	})
}

type updateCommand struct {
	ref    string
	oldSHA string
	newSHA string
}

func (c *updateCommand) run(*kingpin.ParseContext) error {
	return run(func(ctx context.Context, hook *GitHook) error {
		return hook.Update(ctx, c.ref, c.oldSHA, c.newSHA)
	})
}

type postReceiveCommand struct{}

func (c *postReceiveCommand) run(*kingpin.ParseContext) error {
	return run(func(ctx context.Context, hook *GitHook) error {
		return hook.PostReceive(ctx)
	})
}

func run(fn func(ctx context.Context, hook *GitHook) error) error {
	// load hook here (as it loads environment variables, has to be done at time of execution, not register)
	hook, err := NewFromEnvironment()
	if err != nil {
		if errors.Is(err, ErrHookDisabled) {
			return nil
		}
		return err
	}

	// Create context that listens for the interrupt signal from the OS and has a timeout.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, ExecutionTimeout)
	defer cancel()

	return fn(ctx, hook)
}
