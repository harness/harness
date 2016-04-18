package restapi

import (
	"net/http"

	errors "github.com/go-swagger/go-swagger/errors"
	httpkit "github.com/go-swagger/go-swagger/httpkit"
	middleware "github.com/go-swagger/go-swagger/httpkit/middleware"

	"github.com/go-swagger/go-swagger/examples/task-tracker/restapi/operations"
	"github.com/go-swagger/go-swagger/examples/task-tracker/restapi/operations/tasks"
)

// This file is safe to edit. Once it exists it will not be overwritten

func configureAPI(api *operations.TaskTrackerAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	api.JSONConsumer = httpkit.JSONConsumer()

	api.JSONProducer = httpkit.JSONProducer()

	api.TokenHeaderAuth = func(token string) (interface{}, error) {
		return nil, errors.NotImplemented("api key auth (token_header) X-Token from header has not yet been implemented")
	}

	api.APIKeyAuth = func(token string) (interface{}, error) {
		return nil, errors.NotImplemented("api key auth (api_key) token from query has not yet been implemented")
	}

	api.TasksAddCommentToTaskHandler = tasks.AddCommentToTaskHandlerFunc(func(params tasks.AddCommentToTaskParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation tasks.AddCommentToTask has not yet been implemented")
	})
	api.TasksCreateTaskHandler = tasks.CreateTaskHandlerFunc(func(params tasks.CreateTaskParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation tasks.CreateTask has not yet been implemented")
	})
	api.TasksDeleteTaskHandler = tasks.DeleteTaskHandlerFunc(func(params tasks.DeleteTaskParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation tasks.DeleteTask has not yet been implemented")
	})
	api.TasksGetTaskCommentsHandler = tasks.GetTaskCommentsHandlerFunc(func(params tasks.GetTaskCommentsParams) middleware.Responder {
		return middleware.NotImplemented("operation tasks.GetTaskComments has not yet been implemented")
	})
	api.TasksGetTaskDetailsHandler = tasks.GetTaskDetailsHandlerFunc(func(params tasks.GetTaskDetailsParams) middleware.Responder {
		return middleware.NotImplemented("operation tasks.GetTaskDetails has not yet been implemented")
	})
	api.TasksListTasksHandler = tasks.ListTasksHandlerFunc(func(params tasks.ListTasksParams) middleware.Responder {
		return middleware.NotImplemented("operation tasks.ListTasks has not yet been implemented")
	})
	api.TasksUpdateTaskHandler = tasks.UpdateTaskHandlerFunc(func(params tasks.UpdateTaskParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation tasks.UpdateTask has not yet been implemented")
	})
	api.TasksUploadTaskFileHandler = tasks.UploadTaskFileHandlerFunc(func(params tasks.UploadTaskFileParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation tasks.UploadTaskFile has not yet been implemented")
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
