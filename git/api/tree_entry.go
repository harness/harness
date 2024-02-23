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
	"strconv"
)

// EntryMode the type of the object in the git tree.
type EntryMode int

// There are only a few file modes in Git. They look like unix file modes, but they can only be
// one of these.
const (
	EntryModeBlob    EntryMode = 0o100644
	EntryModeExec    EntryMode = 0o100755
	EntryModeSymlink EntryMode = 0o120000
	EntryModeCommit  EntryMode = 0o160000
	EntryModeTree    EntryMode = 0o040000
)

// String converts an EntryMode to a string.
func (e EntryMode) String() string {
	return strconv.FormatInt(int64(e), 8)
}

// ToEntryMode converts a string to an EntryMode.
func ToEntryMode(value string) EntryMode {
	v, _ := strconv.ParseInt(value, 8, 32)
	return EntryMode(v)
}

// TreeEntry the leaf in the git tree.
type TreeEntry struct {
	ID SHA

	entryMode EntryMode
	name      string

	size     int64
	sized    bool
	fullName string
}

// Name returns the name of the entry.
func (e *TreeEntry) Name() string {
	if e.fullName != "" {
		return e.fullName
	}
	return e.name
}

// Mode returns the mode of the entry.
func (e *TreeEntry) Mode() EntryMode {
	return e.entryMode
}

// IsSubModule if the entry is a sub module.
func (e *TreeEntry) IsSubModule() bool {
	return e.entryMode == EntryModeCommit
}

// IsDir if the entry is a sub dir.
func (e *TreeEntry) IsDir() bool {
	return e.entryMode == EntryModeTree
}

// IsLink if the entry is a symlink.
func (e *TreeEntry) IsLink() bool {
	return e.entryMode == EntryModeSymlink
}

// IsRegular if the entry is a regular file.
func (e *TreeEntry) IsRegular() bool {
	return e.entryMode == EntryModeBlob
}

// IsExecutable if the entry is an executable file (not necessarily binary).
func (e *TreeEntry) IsExecutable() bool {
	return e.entryMode == EntryModeExec
}
