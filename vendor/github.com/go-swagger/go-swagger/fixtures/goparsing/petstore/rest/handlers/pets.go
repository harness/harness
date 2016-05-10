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

package handlers

import (
	"net/http"

	"github.com/go-swagger/go-swagger/fixtures/goparsing/petstore/models"
	"github.com/naoina/denco"
)

// A GenericError is the default error message that is generated.
// For certain status codes there are more appropriate error structures.
//
// swagger:response genericError
type GenericError struct {
	// in: body
	Body struct {
		Code    int32 `json:"code"`
		Message error `json:"message"`
	} `json:"body"`
}

// A ValidationError is an that is generated for validation failures.
// It has the same fields as a generic error but adds a Field property.
//
// swagger:response validationError
type ValidationError struct {
	// in: body
	Body struct {
		Code    int32  `json:"code"`
		Message string `json:"message"`
		Field   string `json:"field"`
	} `json:"body"`
}

// A PetQueryFlags contains the query flags for things that list pets.
// swagger:parameters listPets
type PetQueryFlags struct {
	Status string `json:"status"`
}

// A PetID parameter model.
//
// This is used for operations that want the ID of an pet in the path
// swagger:parameters getPetById deletePet updatePet
type PetID struct {
	// The ID of the pet
	//
	// in: path
	// required: true
	ID int64 `json:"id"`
}

// A PetBodyParams model.
//
// This is used for operations that want an Order as body of the request
// swagger:parameters updatePet createPet
type PetBodyParams struct {
	// The pet to submit.
	//
	// in: body
	// required: true
	Pet *models.Pet `json:"pet"`
}

// GetPets swagger:route GET /pets pets listPets
//
// Lists the pets known to the store.
//
// By default it will only lists pets that are available for sale.
// This can be changed with the status flag.
//
// Responses:
// 		default: genericError
// 		    200: []pet
func GetPets(w http.ResponseWriter, r *http.Request, params denco.Params) {
	// some actual stuff should happen in here
}

// GetPetByID swagger:route GET /pets/{id} pets getPetById
//
// Gets the details for a pet.
//
// Responses:
//    default: genericError
//        200: pet
func GetPetByID(w http.ResponseWriter, r *http.Request, params denco.Params) {
	// some actual stuff should happen in here
}

// CreatePet swagger:route POST /pets pets createPet
//
// Creates a new pet in the store.
//
// Responses:
//    default: genericError
//        200: pet
//        422: validationError
func CreatePet(w http.ResponseWriter, r *http.Request, params denco.Params) {
	// some actual stuff should happen in here
}

// UpdatePet swagger:route PUT /pets/{id} pets updatePet
//
// Updates the details for a pet.
//
// Responses:
//    default: genericError
//        200: pet
//        422: validationError
func UpdatePet(w http.ResponseWriter, r *http.Request, params denco.Params) {
	// some actual stuff should happen in here
}

// DeletePet swagger:route DELETE /pets/{id} pets deletePet
//
// Deletes a pet from the store.
//
// Responses:
//    default: genericError
//        204:
func DeletePet(w http.ResponseWriter, r *http.Request, params denco.Params) {
	// some actual stuff should happen in here
}
