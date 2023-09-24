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

package gitea

import (
	"errors"
	"os/exec"

	"code.gitea.io/gitea/modules/git"
)

type runStdError struct {
	err    error
	stderr string
	errMsg string
}

func (r *runStdError) Error() string {
	// the stderr must be in the returned error text, some code only checks `strings.Contains(err.Error(), "git error")`
	if r.errMsg == "" {
		r.errMsg = git.ConcatenateError(r.err, r.stderr).Error()
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
