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

package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api/foreachref"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/hook"

	"github.com/rs/zerolog/log"
)

func DefaultInstructor(
	_ WalkReferencesEntry,
) (WalkInstruction, error) {
	return WalkInstructionHandle, nil
}

// WalkReferences uses the provided options to filter the available references of the repo,
// and calls the handle function for every matching node.
// The instructor & handler are called with a map that contains the matching value for every field provided in fields.
// TODO: walkReferences related code should be moved to separate file.
func (g *Git) WalkReferences(
	ctx context.Context,
	repoPath string,
	handler WalkReferencesHandler,
	opts *WalkReferencesOptions,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	// backfil optional options
	if opts.Instructor == nil {
		opts.Instructor = DefaultInstructor
	}
	if len(opts.Fields) == 0 {
		opts.Fields = []GitReferenceField{GitReferenceFieldRefName, GitReferenceFieldObjectName}
	}
	if opts.MaxWalkDistance <= 0 {
		opts.MaxWalkDistance = math.MaxInt32
	}
	if opts.Patterns == nil {
		opts.Patterns = []string{}
	}
	if string(opts.Sort) == "" {
		opts.Sort = GitReferenceFieldRefName
	}

	// prepare for-each-ref input
	sortArg := mapToReferenceSortingArgument(opts.Sort, opts.Order)
	rawFields := make([]string, len(opts.Fields))
	for i := range opts.Fields {
		rawFields[i] = string(opts.Fields[i])
	}
	format := foreachref.NewFormat(rawFields...)

	// initializer pipeline for output processing
	pipeOut, pipeIn := io.Pipe()
	defer pipeOut.Close()

	go func() {
		cmd := command.New("for-each-ref",
			command.WithFlag("--format", format.Flag()),
			command.WithFlag("--sort", sortArg),
			command.WithFlag("--count", strconv.Itoa(int(opts.MaxWalkDistance))),
			command.WithFlag("--ignore-case"),
		)
		cmd.Add(command.WithArg(opts.Patterns...))
		err := cmd.Run(ctx,
			command.WithDir(repoPath),
			command.WithStdout(pipeIn),
		)
		if err != nil {
			_ = pipeIn.CloseWithError(err)
		} else {
			_ = pipeIn.Close()
		}
	}()

	// TODO: return error from git command!!!!

	parser := format.Parser(pipeOut)
	return walkReferenceParser(parser, handler, opts)
}

func walkReferenceParser(
	parser *foreachref.Parser,
	handler WalkReferencesHandler,
	opts *WalkReferencesOptions,
) error {
	for i := int32(0); i < opts.MaxWalkDistance; i++ {
		// parse next line - nil if end of output reached or an error occurred.
		rawRef := parser.Next()
		if rawRef == nil {
			break
		}

		// convert to correct map.
		ref, err := mapRawRef(rawRef)
		if err != nil {
			return err
		}

		// check with the instructor on the next instruction.
		instruction, err := opts.Instructor(ref)
		if err != nil {
			return fmt.Errorf("error getting instruction: %w", err)
		}

		if instruction == WalkInstructionSkip {
			continue
		}
		if instruction == WalkInstructionStop {
			break
		}

		// otherwise handle the reference.
		err = handler(ref)
		if err != nil {
			return fmt.Errorf("error handling reference: %w", err)
		}
	}

	if err := parser.Err(); err != nil {
		return processGitErrorf(err, "failed to parse reference walk output")
	}

	return nil
}

// GetRef get's the target of a reference
// IMPORTANT provide full reference name to limit risk of collisions across reference types
// (e.g `refs/heads/main` instead of `main`).
func (g *Git) GetRef(
	ctx context.Context,
	repoPath string,
	ref string,
) (string, error) {
	if repoPath == "" {
		return "", ErrRepositoryPathEmpty
	}
	cmd := command.New("show-ref",
		command.WithFlag("--verify"),
		command.WithFlag("-s"),
		command.WithPostSepArg(ref),
	)
	output := &bytes.Buffer{}
	err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(output))
	if err != nil {
		if command.AsError(err).IsExitCode(128) && strings.Contains(err.Error(), "not a valid ref") {
			return "", errors.NotFound("reference %q not found", ref)
		}
		return "", err
	}

	return strings.TrimSpace(output.String()), nil
}

// UpdateRef allows to update / create / delete references
// IMPORTANT provide full reference name to limit risk of collisions across reference types
// (e.g `refs/heads/main` instead of `main`).
func (g *Git) UpdateRef(
	ctx context.Context,
	envVars map[string]string,
	repoPath string,
	ref string,
	oldValue string,
	newValue string,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}

	// don't break existing interface - user calls with empty value to delete the ref.
	if newValue == "" {
		newValue = NilSHA
	}

	// if no old value was provided, use current value (as required for hooks)
	// TODO: technically a delete could fail if someone updated the ref in the meanwhile.
	//nolint:gocritic,nestif
	if oldValue == "" {
		val, err := g.GetRef(ctx, repoPath, ref)
		if errors.IsNotFound(err) {
			// fail in case someone tries to delete a reference that doesn't exist.
			if newValue == NilSHA {
				return errors.NotFound("reference %q not found", ref)
			}

			oldValue = NilSHA
		} else if err != nil {
			return fmt.Errorf("failed to get current value of reference: %w", err)
		} else {
			oldValue = val
		}
	}

	err := g.updateRefWithHooks(
		ctx,
		envVars,
		repoPath,
		ref,
		oldValue,
		newValue,
	)
	if err != nil {
		return fmt.Errorf("failed to update reference with hooks: %w", err)
	}

	return nil
}

// updateRefWithHooks performs a git-ref update for the provided reference.
// Requires both old and new value to be provided explcitly, or the call fails (ensures consistency across operation).
// pre-receice will be called before the update, post-receive after.
func (g *Git) updateRefWithHooks(
	ctx context.Context,
	envVars map[string]string,
	repoPath string,
	ref string,
	oldValue string,
	newValue string,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}

	if oldValue == "" {
		return fmt.Errorf("oldValue can't be empty")
	}
	if newValue == "" {
		return fmt.Errorf("newValue can't be empty")
	}
	if oldValue == NilSHA && newValue == NilSHA {
		return fmt.Errorf("provided values cannot be both empty")
	}

	githookClient, err := g.githookFactory.NewClient(ctx, envVars)
	if err != nil {
		return fmt.Errorf("failed to create githook client: %w", err)
	}

	// call pre-receive before updating the reference
	out, err := githookClient.PreReceive(ctx, hook.PreReceiveInput{
		RefUpdates: []hook.ReferenceUpdate{
			{
				Ref: ref,
				Old: oldValue,
				New: newValue,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("pre-receive call failed with: %w", err)
	}
	if out.Error != nil {
		return fmt.Errorf("pre-receive call returned error: %q", *out.Error)
	}

	if g.traceGit {
		log.Ctx(ctx).Trace().
			Str("git", "pre-receive").
			Msgf("pre-receive call succeeded with output:\n%s", strings.Join(out.Messages, "\n"))
	}

	cmd := command.New("update-ref")
	if newValue == NilSHA {
		cmd.Add(command.WithFlag("-d", ref))
	} else {
		cmd.Add(command.WithArg(ref, newValue))
	}

	cmd.Add(command.WithArg(oldValue))
	err = cmd.Run(ctx, command.WithDir(repoPath))
	if err != nil {
		return processGitErrorf(err, "update of ref %q from %q to %q failed", ref, oldValue, newValue)
	}

	// call post-receive after updating the reference
	out, err = githookClient.PostReceive(ctx, hook.PostReceiveInput{
		RefUpdates: []hook.ReferenceUpdate{
			{
				Ref: ref,
				Old: oldValue,
				New: newValue,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("post-receive call failed with: %w", err)
	}
	if out.Error != nil {
		return fmt.Errorf("post-receive call returned error: %q", *out.Error)
	}

	if g.traceGit {
		log.Ctx(ctx).Trace().
			Str("git", "post-receive").
			Msgf("post-receive call succeeded with output:\n%s", strings.Join(out.Messages, "\n"))
	}

	return nil
}
