package gitrpc

import "github.com/google/wire"

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideClient,
)

func ProvideClient() (Interface, error) {
	return InitClient("localhost:5001")
}
