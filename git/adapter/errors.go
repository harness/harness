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
	"os/exec"
	"strings"

	"github.com/harness/gitness/errors"

	gitea "code.gitea.io/gitea/modules/git"
	"github.com/rs/zerolog/log"
)

var (
	ErrRepositoryPathEmpty = errors.InvalidArgument("repository path cannot be empty")
	ErrBranchNameEmpty     = errors.InvalidArgument("branch name cannot be empty")
	ErrParseDiffHunkHeader = errors.Internal(nil, "failed to parse diff hunk header")
)

type runStdError struct {
	err    error
	stderr string
	errMsg string
}

func (r *runStdError) Error() string {
	// the stderr must be in the returned error text, some code only checks `strings.Contains(err.Error(), "git error")`
	if r.errMsg == "" {
		r.errMsg = gitea.ConcatenateError(r.err, r.stderr).Error()
	}
	return r.errMsg
}

func (r *runStdError) Unwrap() error {
	return r.err
}

func (r *runStdError) Stderr() string {
	return r.stderr
}

func (r *runStdError) IsExitCode(code int) bool {
	var exitError *exec.ExitError
	if errors.As(r.err, &exitError) {
		return exitError.ExitCode() == code
	}
	return false
}

// Logs the error and message, returns either the provided message or a git equivalent if possible.
// Always logs the full message with error as warning.
func processGiteaErrorf(err error, format string, args ...interface{}) error {
	// create fallback error returned if we can't map it
	fallbackErr := errors.Internal(err, format, args...)

	// always log internal error together with message.
	log.Warn().Msgf("%v: [GITEA] %v", fallbackErr, err)

	// check if it's a RunStdError error (contains raw git error)
	var runStdErr gitea.RunStdError
	if errors.As(err, &runStdErr) {
		return mapGiteaRunStdError(runStdErr, fallbackErr)
	}

	switch {
	// gitea is using errors.New(no such file or directory") exclusively for OpenRepository ... (at least as of now)
	case err.Error() == "no such file or directory":
		return errors.NotFound("repository not found")
	case gitea.IsErrNotExist(err):
		return errors.NotFound(format, args, err)
	case gitea.IsErrBranchNotExist(err):
		return errors.NotFound(format, args, err)
	default:
		return fallbackErr
	}
}

// TODO: Improve gitea error handling.
// Doubt this will work for all std errors, as git doesn't seem to have nice error codes.
func mapGiteaRunStdError(err gitea.RunStdError, fallback error) error {
	switch {
	// exit status 128 - fatal: A branch named 'mybranch' already exists.
	// exit status 128 - fatal: cannot lock ref 'refs/heads/a': 'refs/heads/a/b' exists; cannot create 'refs/heads/a'
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "exists"):
		return errors.Conflict(err.Stderr())

	// exit status 128 - fatal: 'a/bc/d/' is not a valid branch name.
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "not a valid"):
		return errors.InvalidArgument(err.Stderr())

	// exit status 1 - error: branch 'mybranch' not found.
	case err.IsExitCode(1) && strings.Contains(err.Stderr(), "not found"):
		return errors.NotFound(err.Stderr())

	// exit status 128 - fatal: ambiguous argument 'branch1...branch2': unknown revision or path not in the working tree.
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "unknown revision"):
		msg := "unknown revision or path not in the working tree"
		// parse the error response from git output
		lines := strings.Split(err.Error(), "\n")
		if len(lines) > 0 {
			cols := strings.Split(lines[0], ": ")
			if len(cols) >= 2 {
				msg = cols[1] + ", " + cols[2]
			}
		}
		return errors.NotFound(msg)

	// exit status 128 - fatal: couldn't find remote ref v1.
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "couldn't find"):
		return errors.NotFound(err.Stderr())

	// exit status 128 - fatal: unable to access 'http://127.0.0.1:4101/hvfl1xj5fojwlrw77xjflw80uxjous254jrr967rvj/':
	//   Failed to connect to 127.0.0.1 port 4101 after 4 ms: Connection refused
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "Failed to connect"):
		return errors.Internal(err, "failed to connect")

	default:
		return fallback
	}
}
