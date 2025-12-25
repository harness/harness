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

const (
	GitCommitterName  = "GIT_COMMITTER_NAME"
	GitCommitterEmail = "GIT_COMMITTER_EMAIL"
	GitCommitterDate  = "GIT_COMMITTER_DATE"
	GitAuthorName     = "GIT_AUTHOR_NAME"
	GitAuthorEmail    = "GIT_AUTHOR_EMAIL"
	GitAuthorDate     = "GIT_AUTHOR_DATE"

	GitTrace            = "GIT_TRACE"
	GitTracePack        = "GIT_TRACE_PACK_ACCESS"
	GitTracePackAccess  = "GIT_TRACE_PACKET"
	GitTracePerformance = "GIT_TRACE_PERFORMANCE"
	GitTraceSetup       = "GIT_TRACE_SETUP"
	GitExecPath         = "GIT_EXEC_PATH" // tells Git where to find its binaries.

	GitObjectDir           = "GIT_OBJECT_DIRECTORY"
	GitAlternateObjectDirs = "GIT_ALTERNATE_OBJECT_DIRECTORIES"
)

// Envs custom key value store for environment variables.
type Envs map[string]string

func (e Envs) Args() []string {
	slice := make([]string, 0, len(e))
	for key, val := range e {
		slice = append(slice, key+"="+val)
	}
	return slice
}
