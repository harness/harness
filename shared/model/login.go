package model

// Login represents a standard subset of user meta-data
// provided by OAuth login services.
type Login struct {
	ID     int64
	Login  string
	Access string
	Secret string
	Name   string
	Email  string
	Expiry int64
}
