package makeplans

// A Booking in the system
type Booking struct {
	// ID the id of the booking
	//
	// required: true
	// read only: true
	ID int64 `json:"id,omitempty"`

	// Subject the subject of this booking
	// required: true
	Subject string
}
