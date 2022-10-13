// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"github.com/harness/gitness/internal/router"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(ProvideServer)

// ProvideServer provides a server instance.
func ProvideServer(config *types.Config, router *router.Router) *Server {
	return &Server{
		Acme:   config.Server.Acme.Enabled,
		Addr:   config.Server.Bind,
		Host:   config.Server.Host,
		router: router,
	}
}
