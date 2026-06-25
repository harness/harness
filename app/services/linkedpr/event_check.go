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

package linkedpr

// CheckPayload carries one or more status checks from an external CI provider.
// Each CheckEntry maps to a single upsert into the gitness CheckStore.
type CheckPayload struct {
	Checks     []CheckEntry
	Repository Repository
}

// CheckEntry is the SCM-agnostic representation of a single check_run / status event.
type CheckEntry struct {
	Identifier string
	Status     string
	Conclusion string
	Link       string
	SHA        string
	Started    string
	Completed  string
}

func (CheckPayload) Kind() Kind               { return KindCheck }
func (p CheckPayload) RepoProviderID() string { return p.Repository.ProviderID }
