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

import (
	"strings"
)

// RepoAttr defines repo attributes that can be used for sorting and filtering.
type RepoAttr int

// RepoAttr enumeration.
const (
	RepoAttrNone RepoAttr = iota
	// TODO [CODE-1363]: remove after identifier migration.
	RepoAttrUID
	RepoAttrIdentifier
	RepoAttrCreated
	RepoAttrUpdated
	RepoAttrDeleted
)

// ParseRepoAttr parses the repo attribute string
// and returns the equivalent enumeration.
func ParseRepoAttr(s string) RepoAttr {
	switch strings.ToLower(s) {
	// TODO [CODE-1363]: remove after identifier migration.
	case uid:
		return RepoAttrUID
	case identifier:
		return RepoAttrIdentifier
	case created, createdAt:
		return RepoAttrCreated
	case updated, updatedAt:
		return RepoAttrUpdated
	case deleted, deletedAt:
		return RepoAttrDeleted
	default:
		return RepoAttrNone
	}
}

// String returns the string representation of the attribute.
func (a RepoAttr) String() string {
	switch a {
	// TODO [CODE-1363]: remove after identifier migration.
	case RepoAttrUID:
		return uid
	case RepoAttrIdentifier:
		return identifier
	case RepoAttrCreated:
		return created
	case RepoAttrUpdated:
		return updated
	case RepoAttrDeleted:
		return deleted
	case RepoAttrNone:
		return ""
	default:
		return undefined
	}
}

// RepoState defines repo state.
type RepoState int

// RepoState enumeration.
const (
	RepoStateActive RepoState = iota
	RepoStateGitImport
	RepoStateMigrateGitPush
	RepoStateMigrateDataImport
)

// String returns the string representation of the RepoState.
func (state RepoState) String() string {
	switch state {
	case RepoStateActive:
		return "active"
	case RepoStateGitImport:
		return "git-import"
	case RepoStateMigrateGitPush:
		return "migrate-git-push"
	case RepoStateMigrateDataImport:
		return "migrate-data-import"
	default:
		return undefined
	}
}
