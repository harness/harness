package internal

import (
	"fmt"
)

type APIClientErr struct {
	Message string
	URL     string
	Cause   error
}

func (e APIClientErr) Error() string {
	return fmt.Sprintf("%s (Requested %s): %v", e.Message, e.URL, e.Cause)
}
