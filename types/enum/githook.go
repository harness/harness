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

package enum

// GitOpType represents the type of operation triggering the githook.
type GitOpType string

// GitOpType enumeration.
const (
	// GitOpTypeGitPush represents direct git pushes from git clients.
	GitOpTypeGitPush GitOpType = "git_push"

	// GitOpTypeAPIRefsOnly represents branch/tag management and PR merge operations via API.
	GitOpTypeAPIRefsOnly GitOpType = "api_refs_only"

	// GitOpTypeAPISystemRefs represents internal system-maintained reference operations.
	GitOpTypeAPISystemRefs GitOpType = "api_system_refs"

	// GitOpTypeAPILinkedSync represents linked-repository synchronization operations.
	GitOpTypeAPILinkedSync GitOpType = "api_linked_sync"

	// GitOpTypeAPIContent represents commit API and apply comment suggestions operations.
	GitOpTypeAPIContent GitOpType = "api_content"

	// GitOpTypeAPIContentBypassRules represents commit API with bypass.
	GitOpTypeAPIContentBypassRules GitOpType = "api_content_bypass_rules"

	// GitOpTypeManageRepo represents repo lifecycle operations (create, delete, sync/import/link).
	GitOpTypeManageRepo GitOpType = "manage_repo"

	// GitOpTypeMergeQueue represents internal system-maintained reference operations of merge queue service.
	GitOpTypeMergeQueue GitOpType = "merge_queue"
)

var gitOpTypes = sortEnum([]GitOpType{
	GitOpTypeGitPush,
	GitOpTypeAPIRefsOnly,
	GitOpTypeAPISystemRefs,
	GitOpTypeAPILinkedSync,
	GitOpTypeAPIContent,
	GitOpTypeAPIContentBypassRules,
	GitOpTypeManageRepo,
	GitOpTypeMergeQueue,
})

func (GitOpType) Enum() []any {
	return toInterfaceSlice(gitOpTypes)
}

func (s GitOpType) Sanitize() (GitOpType, bool) {
	return Sanitize(s, GetAllGitOpTypes)
}

func GetAllGitOpTypes() ([]GitOpType, GitOpType) {
	return gitOpTypes, GitOpTypeGitPush
}
