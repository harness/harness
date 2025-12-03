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

type LFSObject struct {
	ID        int64  `json:"id"`
	OID       string `json:"oid"`
	Size      int64  `json:"size"`
	Created   int64  `json:"created"`
	CreatedBy int64  `json:"created_by"`
	RepoID    int64  `json:"repo_id"`
}

type LFSLock struct {
	ID      int64  `json:"id"`
	Path    string `json:"path"`
	Ref     string `json:"ref"`
	Created int64  `json:"created"`
	RepoID  int64  `json:"repo_id"`
}
