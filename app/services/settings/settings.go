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

package settings

type Key string

var (
	// KeySecretScanningEnabled [bool] enables secret scanning if set to true.
	KeySecretScanningEnabled       Key = "secret_scanning_enabled"
	DefaultSecretScanningEnabled       = false
	KeyFileSizeLimit               Key = "file_size_limit"
	DefaultFileSizeLimit               = int64(1e+8) // 100 MB
	KeyInstallID                   Key = "install_id"
	DefaultInstallID                   = string("")
	KeyPrincipalCommitterMatch     Key = "principal_committer_match"
	DefaultPrincipalCommitterMatch     = false
	KeyGitLFSEnabled               Key = "git_lfs_enabled"
	DefaultGitLFSEnabled               = true
	KeyAutoMergeEnabled            Key = "auto_merge_enabled"
	DefaultAutoMergeEnabled            = false
	DefaultBranchKey               Key = "default_branch"
	DefaultBranch                      = string("main")

	// KeyAIAgentsKillSwitch [bool] disables autonomous agents for a repo if set to true.
	KeyAIAgentsKillSwitch     Key = "ai_agents_kill_switch"
	DefaultAIAgentsKillSwitch     = false
	// KeyAIAgentsAutonomy [int] is the agent autonomy level (0 observe, 1 suggest, 2 draft PR, 3 auto-merge).
	KeyAIAgentsAutonomy     Key = "ai_agents_autonomy"
	DefaultAIAgentsAutonomy     = AutonomyDraftPR
	// KeyAIAgentsMaxActive [int] is the maximum number of simultaneously active agents for a repo.
	KeyAIAgentsMaxActive     Key = "ai_agents_max_active"
	DefaultAIAgentsMaxActive     = 2
)
