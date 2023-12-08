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

package types

import (
	"time"

	"github.com/harness/gitness/git/enum"
)

// Config defines the configuration for the git package.
type Config struct {
	// Trace specifies whether git operations' stdIn/stdOut/err are traced.
	// NOTE: Currently limited to 'push' operation until we move to internal command package.
	Trace bool
	// Root specifies the directory containing git related data (e.g. repos, ...)
	Root string
	// TmpDir (optional) specifies the directory for temporary data (e.g. repo clones, ...)
	TmpDir string
	// HookPath points to the binary used as git server hook.
	HookPath string

	// LastCommitCache holds configuration options for the last commit cache.
	LastCommitCache LastCommitCacheConfig
}

// LastCommitCacheConfig holds configuration options for the last commit cache.
type LastCommitCacheConfig struct {
	// Mode determines where the cache will be.
	Mode enum.LastCommitCacheMode

	// Duration defines cache duration of last commit.
	Duration time.Duration
}
