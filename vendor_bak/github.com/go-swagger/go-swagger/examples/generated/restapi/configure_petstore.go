package restapi

import (
	"io"
	"net/http"

	errors "github.com/go-swagger/go-swagger/errors"
	httpkit "github.com/go-swagger/go-swagger/httpkit"
	middleware "github.com/go-swagger/go-swagger/httpkit/middleware"

	"github.com/go-swagger/go-swagger/examples/generated/restapi/operations"
	"github.com/go-swagger/go-swagger/examples/generated/restapi/operations/pet"
	"github.com/go-swagger/go-swagger/examples/generated/restapi/operations/store"
	"github.com/go-swagger/go-swagger/examples/generated/restapi/operations/user"
)

// This file is safe to edit. Once it exists it will not be overwritten

func configureAPI(api *operations.PetstoreAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	api.JSONConsumer = httpkit.JSONConsumer()

	api.XMLConsumer = httpkit.ConsumerFunc(func(r io.Reader, target interface{}) error {
		return errors.NotImplemented("xml consumer has not yet been implemented")
	})

	api.JSONProducer = httpkit.JSONProducer()

	api.XMLProducer = httpkit.ProducerFunc(func(w io.Writer, data interface{}) error {
		return errors.NotImplemented("xml producer has not yet been implemented")
	})

	api.APIKeyAuth = func(token string) (interface{}, error) {
		return nil, errors.NotImplemented("api key auth (api_key) api_key from header has not yet been implemented")
	}

	api.PetAddPetHandler = pet.AddPetHandlerFunc(func(params pet.AddPetParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation pet.AddPet has not yet been implemented")
	})
	api.UserCreateUserHandler = user.CreateUserHandlerFunc(func(params user.CreateUserParams) middleware.Responder {
		return middleware.NotImplemented("operation user.CreateUser has not yet been implemented")
	})
	api.UserCreateUsersWithArrayInputHandler = user.CreateUsersWithArrayInputHandlerFunc(func(params user.CreateUsersWithArrayInputParams) middleware.Responder {
		return middleware.NotImplemented("operation user.CreateUsersWithArrayInput has not yet been implemented")
	})
	api.UserCreateUsersWithListInputHandler = user.CreateUsersWithListInputHandlerFunc(func(params user.CreateUsersWithListInputParams) middleware.Responder {
		return middleware.NotImplemented("operation user.CreateUsersWithListInput has not yet been implemented")
	})
	api.StoreDeleteOrderHandler = store.DeleteOrderHandlerFunc(func(params store.DeleteOrderParams) middleware.Responder {
		return middleware.NotImplemented("operation store.DeleteOrder has not yet been implemented")
	})
	api.PetDeletePetHandler = pet.DeletePetHandlerFunc(func(params pet.DeletePetParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation pet.DeletePet has not yet been implemented")
	})
	api.UserDeleteUserHandler = user.DeleteUserHandlerFunc(func(params user.DeleteUserParams) middleware.Responder {
		return middleware.NotImplemented("operation user.DeleteUser has not yet been implemented")
	})
	api.PetFindPetsByStatusHandler = pet.FindPetsByStatusHandlerFunc(func(params pet.FindPetsByStatusParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation pet.FindPetsByStatus has not yet been implemented")
	})
	api.PetFindPetsByTagsHandler = pet.FindPetsByTagsHandlerFunc(func(params pet.FindPetsByTagsParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation pet.FindPetsByTags has not yet been implemented")
	})
	api.StoreGetOrderByIDHandler = store.GetOrderByIDHandlerFunc(func(params store.GetOrderByIDParams) middleware.Responder {
		return middleware.NotImplemented("operation store.GetOrderByID has not yet been implemented")
	})
	api.PetGetPetByIDHandler = pet.GetPetByIDHandlerFunc(func(params pet.GetPetByIDParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation pet.GetPetByID has not yet been implemented")
	})
	api.UserGetUserByNameHandler = user.GetUserByNameHandlerFunc(func(params user.GetUserByNameParams) middleware.Responder {
		return middleware.NotImplemented("operation user.GetUserByName has not yet been implemented")
	})
	api.UserLoginUserHandler = user.LoginUserHandlerFunc(func(params user.LoginUserParams) middleware.Responder {
		return middleware.NotImplemented("operation user.LoginUser has not yet been implemented")
	})
	api.UserLogoutUserHandler = user.LogoutUserHandlerFunc(func() middleware.Responder {
		return middleware.NotImplemented("operation user.LogoutUser has not yet been implemented")
	})
	api.StorePlaceOrderHandler = store.PlaceOrderHandlerFunc(func(params store.PlaceOrderParams) middleware.Responder {
		return middleware.NotImplemented("operation store.PlaceOrder has not yet been implemented")
	})
	api.PetUpdatePetHandler = pet.UpdatePetHandlerFunc(func(params pet.UpdatePetParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation pet.UpdatePet has not yet been implemented")
	})
	api.PetUpdatePetWithFormHandler = pet.UpdatePetWithFormHandlerFunc(func(params pet.UpdatePetWithFormParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation pet.UpdatePetWithForm has not yet been implemented")
	})
	api.UserUpdateUserHandler = user.UpdateUserHandlerFunc(func(params user.UpdateUserParams) middleware.Responder {
		return middleware.NotImplemented("operation user.UpdateUser has not yet been implemented")
	})

	api.ServerShutdown = func() {}
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}
