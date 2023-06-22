// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"github.com/harness/gitness/gitrpc/internal/gitea"
	"github.com/harness/gitness/gitrpc/internal/service"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideServer,
	ProvideHTTPServer,
	ProvideGITAdapter,
)

func ProvideGITAdapter() (service.GitAdapter, error) {
	return gitea.New()
}

func ProvideServer(config Config, adapter service.GitAdapter) (*GRPCServer, error) {
	return NewServer(config, adapter)
}

func ProvideHTTPServer(config Config) (*HTTPServer, error) {
	return NewHTTPServer(config)
}
