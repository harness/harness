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

// EntryMode is the unix file mode of a tree entry.
type EntryMode int

// There are only a few file modes in Git. They look like unix file modes, but
// they can only be one of these.
const (
	EntryTree    EntryMode = 0040000
	EntryBlob    EntryMode = 0100644
	EntryExec    EntryMode = 0100755
	EntrySymlink EntryMode = 0120000
	EntryCommit  EntryMode = 0160000
)
