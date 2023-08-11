package encrypt

import (
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideEncrypter,
)

func ProvideEncrypter(config *types.Config) (Encrypter, error) {
	if config.Encrypter.Secret == "" {
		return &none{}, nil
	}
	return New(config.Encrypter.Secret, config.Encrypter.MixedContent)
}
