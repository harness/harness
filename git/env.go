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

package git

import (
	"context"
	"fmt"
)

const (
	EnvActorName  = "GITNESS_HOOK_ACTOR_NAME"
	EnvActorEmail = "GITNESS_HOOK_ACTOR_EMAIL" //#nosec
	EnvRepoUID    = "GITNESS_HOOK_REPO_UID"
	EnvRequestID  = "GITNESS_HOOK_REQUEST_ID"
)

// ASSUMPTION: writeRequst and writeRequst.Actor is never nil.
func CreateEnvironmentForPush(ctx context.Context, writeRequest WriteParams) []string {
	// don't send existing environment variables (os.Environ()), only send what's explicitly necessary.
	// Otherwise we create implicit dependencies that are easy to break.
	environ := []string{
		// request id to use for hooks
		EnvRequestID + "=" + RequestIDFrom(ctx),
		// repo related info
		EnvRepoUID + "=" + writeRequest.RepoUID,
		// actor related info
		EnvActorName + "=" + writeRequest.Actor.Name,
		EnvActorEmail + "=" + writeRequest.Actor.Email,
	}

	// add all environment variables coming from client request
	for key, value := range writeRequest.EnvVars {
		environ = append(environ, fmt.Sprintf("%s=%s", key, value))
	}

	return environ
}
