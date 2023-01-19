// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package services

import (
	"github.com/harness/gitness/internal/services/pullreq"
	"github.com/harness/gitness/internal/services/webhook"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideServices,
)

type Services struct {
	ws  *webhook.Service
	bms *pullreq.Service
}

func ProvideServices(ws *webhook.Service, bms *pullreq.Service) Services {
	return Services{
		ws:  ws,
		bms: bms,
	}
}
