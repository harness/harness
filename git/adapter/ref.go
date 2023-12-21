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

package adapter

import (
	"context"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
	gitearef "code.gitea.io/gitea/modules/git/foreachref"
	"github.com/rs/zerolog/log"
)

func DefaultInstructor(
	_ types.WalkReferencesEntry,
) (types.WalkInstruction, error) {
	return types.WalkInstructionHandle, nil
}

// WalkReferences uses the provided options to filter the available references of the repo,
// and calls the handle function for every matching node.
// The instructor & handler are called with a map that contains the matching value for every field provided in fields.
// TODO: walkGiteaReferences related code should be moved to separate file.
func (a Adapter) WalkReferences(
	ctx context.Context,
	repoPath string,
	handler types.WalkReferencesHandler,
	opts *types.WalkReferencesOptions,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	// backfil optional options
	if opts.Instructor == nil {
		opts.Instructor = DefaultInstructor
	}
	if len(opts.Fields) == 0 {
		opts.Fields = []types.GitReferenceField{types.GitReferenceFieldRefName, types.GitReferenceFieldObjectName}
	}
	if opts.MaxWalkDistance <= 0 {
		opts.MaxWalkDistance = math.MaxInt32
	}
	if opts.Patterns == nil {
		opts.Patterns = []string{}
	}
	if string(opts.Sort) == "" {
		opts.Sort = types.GitReferenceFieldRefName
	}

	// prepare for-each-ref input
	sortArg := mapToGiteaReferenceSortingArgument(opts.Sort, opts.Order)
	rawFields := make([]string, len(opts.Fields))
	for i := range opts.Fields {
		rawFields[i] = string(opts.Fields[i])
	}
	giteaFormat := gitearef.NewFormat(rawFields...)

	// initializer pipeline for output processing
	pipeOut, pipeIn := io.Pipe()
	defer pipeOut.Close()
	defer pipeIn.Close()
	stderr := strings.Builder{}
	rc := &gitea.RunOpts{Dir: repoPath, Stdout: pipeIn, Stderr: &stderr}

	go func() {
		// create array for args as patterns have to be passed as separate args.
		args := []string{
			"for-each-ref",
			"--format",
			giteaFormat.Flag(),
			"--sort",
			sortArg,
			"--count",
			fmt.Sprint(opts.MaxWalkDistance),
			"--ignore-case",
		}
		args = append(args, opts.Patterns...)
		err := gitea.NewCommand(ctx, args...).Run(rc)
		if err != nil {
			_ = pipeIn.CloseWithError(gitea.ConcatenateError(err, stderr.String()))
		} else {
			_ = pipeIn.Close()
		}
	}()

	// TODO: return error from git command!!!!

	parser := giteaFormat.Parser(pipeOut)
	return walkGiteaReferenceParser(parser, handler, opts)
}

func walkGiteaReferenceParser(
	parser *gitearef.Parser,
	handler types.WalkReferencesHandler,
	opts *types.WalkReferencesOptions,
) error {
	for i := int32(0); i < opts.MaxWalkDistance; i++ {
		// parse next line - nil if end of output reached or an error occurred.
		rawRef := parser.Next()
		if rawRef == nil {
			break
		}

		// convert to correct map.
		ref, err := mapGiteaRawRef(rawRef)
		if err != nil {
			return err
		}

		// check with the instructor on the next instruction.
		instruction, err := opts.Instructor(ref)
		if err != nil {
			return fmt.Errorf("error getting instruction: %w", err)
		}

		if instruction == types.WalkInstructionSkip {
			continue
		}
		if instruction == types.WalkInstructionStop {
			break
		}

		// otherwise handle the reference.
		err = handler(ref)
		if err != nil {
			return fmt.Errorf("error handling reference: %w", err)
		}
	}

	if err := parser.Err(); err != nil {
		return processGiteaErrorf(err, "failed to parse reference walk output")
	}

	return nil
}

// GetRef get's the target of a reference
// IMPORTANT provide full reference name to limit risk of collisions across reference types
// (e.g `refs/heads/main` instead of `main`).
func (a Adapter) GetRef(
	ctx context.Context,
	repoPath string,
	ref string,
) (string, error) {
	if repoPath == "" {
		return "", ErrRepositoryPathEmpty
	}
	cmd := gitea.NewCommand(ctx, "show-ref", "--verify", "-s", "--", ref)
	stdout, _, err := cmd.RunStdString(&gitea.RunOpts{
		Dir: repoPath,
	})
	if err != nil {
		if err.IsExitCode(128) && strings.Contains(err.Stderr(), "not a valid ref") {
			return "", types.ErrNotFound("reference %q not found", ref)
		}
		return "", err
	}

	return strings.TrimSpace(stdout), nil
}

// UpdateRef allows to update / create / delete references
// IMPORTANT provide full reference name to limit risk of collisions across reference types
// (e.g `refs/heads/main` instead of `main`).
func (a Adapter) UpdateRef(
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
		newValue = types.NilSHA
	}

	// if no old value was provided, use current value (as required for hooks)
	// TODO: technically a delete could fail if someone updated the ref in the meanwhile.
	//nolint:gocritic,nestif
	if oldValue == "" {
		val, err := a.GetRef(ctx, repoPath, ref)
		if types.IsNotFoundError(err) {
			// fail in case someone tries to delete a reference that doesn't exist.
			if newValue == types.NilSHA {
				return types.ErrNotFound("reference %q not found", ref)
			}

			oldValue = types.NilSHA
		} else if err != nil {
			return fmt.Errorf("failed to get current value of reference: %w", err)
		} else {
			oldValue = val
		}
	}

	err := a.updateRefWithHooks(
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
func (a Adapter) updateRefWithHooks(
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
	if oldValue == types.NilSHA && newValue == types.NilSHA {
		return fmt.Errorf("provided values cannot be both empty")
	}

	githookClient, err := a.githookFactory.NewClient(ctx, envVars)
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

	if a.traceGit {
		log.Ctx(ctx).Trace().
			Str("git", "pre-receive").
			Msgf("pre-receive call succeeded with output:\n%s", strings.Join(out.Messages, "\n"))
	}

	args := make([]string, 0, 4)
	args = append(args, "update-ref")
	if newValue == types.NilSHA {
		args = append(args, "-d", ref)
	} else {
		args = append(args, ref, newValue)
	}

	args = append(args, oldValue)

	cmd := gitea.NewCommand(ctx, args...)
	_, _, err = cmd.RunStdString(&gitea.RunOpts{
		Dir: repoPath,
	})
	if err != nil {
		return processGiteaErrorf(err, "update of ref %q from %q to %q failed", ref, oldValue, newValue)
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

	if a.traceGit {
		log.Ctx(ctx).Trace().
			Str("git", "post-receive").
			Msgf("post-receive call succeeded with output:\n%s", strings.Join(out.Messages, "\n"))
	}

	return nil
}
