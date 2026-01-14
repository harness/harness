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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sync"
)

var (
	GitExecutable = "git"

	actionRegex = regexp.MustCompile(`^[[:alnum:]]+[-[:alnum:]]*$`)
)

// Command contains options for running a git command.
type Command struct {
	// Globals is the number of optional flags to pass before command name.
	// example: git --shallow-file pack-objects ...
	Globals []string

	// Name is the name of the Git command to run, e.g. "log", "cat-file" or "worktree".
	Name string

	// Action is the action of the Git command, e.g. "set-url" in `git remote set-url`
	Action string

	// Flags is the number of optional flags to pass before positional arguments, e.g.
	// `--oneline` or `--format=fuller`.
	Flags []string

	// Args is the arguments that shall be passed after all flags. These arguments must not be
	// flags and thus cannot start with `-`. Note that it may be unsafe to use this field in the
	// case where arguments are directly user-controlled. In that case it is advisable to use
	// `PostSepArgs` instead.
	Args []string

	// PostSepArgs is the arguments that shall be passed as positional arguments after the `--`
	// separator. Git recognizes that separator as the point where it should stop expecting any
	// options and treat the remaining arguments as positionals. This should be used when
	// passing user-controlled input of arbitrary form like for example paths, which may start
	// with a `-`.
	PostSepArgs []string

	// Git environment variables
	Envs Envs

	// internal counter for GIT_CONFIG_COUNT environment variable.
	// more info: [link](https://git-scm.com/docs/git-config#Documentation/git-config.txt-GITCONFIGCOUNT)
	configEnvCounter int

	mux sync.RWMutex
}

// New creates new command for interacting with the git process.
func New(name string, options ...CmdOptionFunc) *Command {
	c := &Command{
		Name: name,
		Envs: make(Envs),
	}

	c.Add(options...)

	return c
}

// Clone clones the command object.
func (c *Command) Clone() *Command {
	c.mux.RLock()
	defer c.mux.RUnlock()

	globals := make([]string, len(c.Globals))
	copy(globals, c.Globals)

	flags := make([]string, len(c.Flags))
	copy(flags, c.Flags)

	args := make([]string, len(c.Args))
	copy(args, c.Args)

	postSepArgs := make([]string, len(c.PostSepArgs))
	copy(postSepArgs, c.Flags)

	envs := make(Envs, len(c.Envs))
	for key, val := range c.Envs {
		envs[key] = val
	}

	return &Command{
		Name:             c.Name,
		Action:           c.Action,
		Flags:            flags,
		Args:             args,
		PostSepArgs:      postSepArgs,
		Envs:             envs,
		configEnvCounter: c.configEnvCounter,
	}
}

// Add appends given options to the command.
func (c *Command) Add(options ...CmdOptionFunc) *Command {
	c.mux.Lock()
	defer c.mux.Unlock()

	for _, opt := range options {
		opt(c)
	}
	return c
}

// Run executes the git command with optional configuration using WithXxx functions.
func (c *Command) Run(ctx context.Context, opts ...RunOptionFunc) (err error) {
	options := &RunOption{}
	for _, f := range opts {
		f(options)
	}

	if options.Stdout == nil {
		options.Stdout = io.Discard
	}
	errAsBuff := false
	if options.Stderr == nil {
		options.Stderr = new(bytes.Buffer)
		errAsBuff = true
	}

	args, err := c.makeArgs()
	if err != nil {
		return fmt.Errorf("failed to build argument list: %w", err)
	}
	cmd := exec.CommandContext(ctx, GitExecutable, args...)
	if len(c.Envs) > 0 {
		cmd.Env = c.Envs.Args()
	}
	cmd.Env = append(cmd.Env, options.Envs...)
	cmd.Dir = options.Dir
	cmd.Stdin = options.Stdin
	cmd.Stdout = options.Stdout
	cmd.Stderr = options.Stderr
	if err = cmd.Start(); err != nil {
		return err
	}

	result := make(chan error)
	go func() {
		result <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		<-result
		if cmd.Process != nil && cmd.ProcessState != nil && !cmd.ProcessState.Exited() {
			if err := cmd.Process.Kill(); err != nil && !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("kill process: %w", err)
			}
		}

		return ctx.Err()
	case err = <-result:
		if err == nil {
			return nil
		}

		var stderr []byte
		if buff, ok := options.Stderr.(*bytes.Buffer); ok && errAsBuff {
			stderr = buff.Bytes()
		}
		return NewError(err, stderr)
	}
}

func (c *Command) makeArgs() ([]string, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	var safeArgs []string

	commandDescription, ok := descriptions[c.Name]
	if !ok {
		return nil, fmt.Errorf("invalid sub command name %q: %w", c.Name, ErrInvalidArg)
	}

	if commandDescription.options != nil {
		options := commandDescription.options()
		for _, option := range options {
			option(c)
		}
	}

	// add globals
	if len(c.Globals) > 0 {
		safeArgs = append(safeArgs, c.Globals...)
	}

	safeArgs = append(safeArgs, c.Name)

	if c.Action != "" {
		if !actionRegex.MatchString(c.Action) {
			return nil, fmt.Errorf("invalid action %q: %w", c.Action, ErrInvalidArg)
		}
		safeArgs = append(safeArgs, c.Action)
	}

	commandArgs, err := commandDescription.args(c.Flags, c.Args, c.PostSepArgs)
	if err != nil {
		return nil, err
	}
	safeArgs = append(safeArgs, commandArgs...)

	return safeArgs, nil
}
