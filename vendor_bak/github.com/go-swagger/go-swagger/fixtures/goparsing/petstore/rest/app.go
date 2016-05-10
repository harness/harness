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

package rest

import (
	"net/http"

	"github.com/go-swagger/go-swagger/fixtures/goparsing/petstore/rest/handlers"
	"github.com/naoina/denco"
)

// ServeAPI serves this api
func ServeAPI() error {
	mux := denco.NewMux()

	routes := []denco.Handler{
		mux.GET("/pets", handlers.GetPets),
		mux.POST("/pets", handlers.CreatePet),
		mux.GET("/pets/:id", handlers.GetPetByID),
		mux.PUT("/pets/:id", handlers.UpdatePet),
		mux.Handler("DELETE", "/pets/:id", handlers.DeletePet),
		mux.GET("/orders/:id", handlers.GetOrderDetails),
		mux.POST("/orders", handlers.CreateOrder),
		mux.PUT("/orders/:id", handlers.UpdateOrder),
		mux.Handler("DELETE", "/orders/:id", handlers.CancelOrder),
	}
	handler, err := mux.Build(routes)
	if err != nil {
		return err
	}
	return http.ListenAndServe(":8000", handler)
}
