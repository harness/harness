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

import "github.com/go-swagger/go-swagger/strfmt"

// An Order for one or more pets by a user.
// swagger:model order
type Order struct {
	// the ID of the order
	//
	// required: true
	ID int64 `json:"id"`

	// the id of the user who placed the order.
	//
	// required: true
	UserID int64 `json:"userId"`

	// the time at which this order was made.
	//
	// required: true
	OrderedAt strfmt.DateTime `json:"orderedAt"`

	// the items for this order
	// mininum items: 1
	Items []struct {

		// the id of the pet to order
		//
		// required: true
		PetID int64 `json:"petId"`

		// the quantity of this pet to order
		//
		// required: true
		// minimum: 1
		Quantity int32 `json:"qty"`
	} `json:"items"`
}
