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

import "github.com/go-swagger/go-swagger/fixtures/goparsing/classification/transitive/mods"

// StoreOrder represents an order in this application.
//
// An order can either be created, processed or completed.
//
// swagger:model order
type StoreOrder struct {
	// the id for this order
	//
	// required: true
	// min: 1
	ID int64 `json:"id"`

	// the name for this user
	//
	// required: true
	// min length: 3
	UserID int64 `json:"userId"`

	// the items for this order
	Items []struct {
		ID       int32    `json:"id"`
		Pet      mods.Pet `json:"pet"`
		Quantity int16    `json:"quantity"`
	} `json:"items"`
}
