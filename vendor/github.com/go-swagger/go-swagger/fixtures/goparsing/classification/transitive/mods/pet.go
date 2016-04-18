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

// Pet represents a pet in our store
//
// this model is not explictly mentioned in the import paths
// but because it it transitively required by the order
// it should also be collected.
//
// swagger:model pet
type Pet struct {
	// ID the id of this pet
	//
	// required: true
	ID int64 `json:"id"`

	// Name the name of the pet
	// this is more like the breed or race of the pet
	//
	// required: true
	// min length: 3
	Name string `json:"name"`

	// Category the category this pet belongs to.
	//
	// required: true
	Category *Category `json:"category"`
}
