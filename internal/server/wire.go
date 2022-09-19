// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"net/http"

	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(ProvideServer)

// ProvideServer provides a server instance.
func ProvideServer(config *types.Config, handler http.Handler) *Server {
	return &Server{
		Acme:    config.Server.Acme.Enabled,
		Addr:    config.Server.Bind,
		Host:    config.Server.Host,
		Handler: handler,
	}
}
