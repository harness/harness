package handler

import (
	"log"
	"net/http"
)

// badRequest is handled by setting the status code in the reply to StatusBadRequest.
type badRequest struct{ error }

// notFound is handled by setting the status code in the reply to StatusNotFound.
type notFound struct{ error }

// notAuthorized is handled by setting the status code in the reply to StatusNotAuthorized.
type notAuthorized struct{ error }

// notImplemented is handled by setting the status code in the reply to StatusNotImplemented.
type notImplemented struct{ error }

// forbidden is handled by setting the status code in the reply to StatusForbidden.
type forbidden struct{ error }

// internalServerError is handled by setting the status code in the reply to StatusInternalServerError.
type internalServerError struct{ error }

// errorHandler wraps a function returning an error by handling the error and returning a http.Handler.
// If the error is of the one of the types defined above, it is handled as described for every type.
// If the error is of another type, it is considered as an internal error and its message is logged.
func errorHandler(f func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// serve the request
		err := f(w, r)
		if err == nil {
			return
		}

		// log the url for debugging purposes
		log.Println(r.Method, r.URL.Path)

		switch err.(type) {
		case badRequest:
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		case notFound:
			http.Error(w, "Not Found", http.StatusNotFound)
		case notAuthorized:
			http.Error(w, "Not Authorized", http.StatusUnauthorized)
		case notImplemented:
			http.Error(w, "Not Implemented", http.StatusForbidden)
		case forbidden:
			http.Error(w, "Forbidden", http.StatusForbidden)
		case internalServerError:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		default:
			log.Println(err)
			http.Error(w, "oops", http.StatusInternalServerError)
		}
	}
}
