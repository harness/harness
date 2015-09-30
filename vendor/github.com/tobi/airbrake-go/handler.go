package airbrake

import (
	"net/http"
)

// CapturePanicHandler "middleware".
// Wraps the http handler so that all panics will be dutifully reported to airbrake
//
// Example:
//   http.HandleFunc("/", airbrake.CapturePanicHandler(MyServerFunc))
func CapturePanicHandler(app http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer CapturePanic(r)
		app(w, r)
	}
}
