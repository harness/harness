// Package booking API.
//
// the purpose of this application is to provide an application
// that is using plain go code to define an API
//
//
//     Schemes: https
//     Host: localhost
//     Version: 0.0.1
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//
// swagger:meta
package booking

import (
	"net/http"

	"github.com/go-swagger/scan-repo-boundary/makeplans"
)

// BookingResponse represents a scheduled appointment
//
// swagger:response BookingResponse
type BookingResponse struct {
	// Booking struct
	//
	// in: body
	// required: true
	Booking makeplans.Booking `json:"booking"`
}

// Bookings swagger:route GET /admin/bookings/ booking Bookings
//
// Bookings lists all the appointments that have been made on the site.
//
//
// Consumes:
// application/json
//
// Schemes: http, https
//
// Produces:
// application/json
//
// Responses:
// 200: BookingResponse
func bookings(w http.ResponseWriter, r *http.Request) {

}
