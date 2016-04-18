// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mods

// NotSelected is a model that is in a transitive package
//
// This model is not annotated and should not be detected for parsing.
type NotSelected struct {
	// ID the id of this not selected model
	ID int64 `json:"id"`
	// Name the name of this not selected model
	Name string `json:"name"`
}

// Notable is a model in a transitive package.
// it's used for embedding in another model
//
// swagger:model withNotes
type Notable struct {
	Notes string `json:"notes"`

	Extra string `json:"extra"`
}
