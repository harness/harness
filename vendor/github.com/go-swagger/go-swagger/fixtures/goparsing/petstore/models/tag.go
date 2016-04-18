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

package models

// A Tag is an extra piece of data to provide more information about a pet.
// It is used to describe the animals available in the store.
// swagger:model tag
type Tag struct {
	// The id of the tag.
	//
	// required: true
	ID int64 `json:"id"`

	// The value of the tag.
	//
	// required: true
	Value string `json:"value"`
}
