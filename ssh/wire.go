// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ssh

import (
	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/services/publickey"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideServer,
)

func ProvideServer(
	config *types.Config,
	vierifier publickey.Service,
	repoctrl *repo.Controller,
) *Server {
	return &Server{
		Host:                    config.SSH.Host,
		Port:                    config.SSH.Port,
		DefaultUser:             config.SSH.DefaultUser,
		Ciphers:                 config.SSH.Ciphers,
		KeyExchanges:            config.SSH.KeyExchanges,
		MACs:                    config.SSH.MACs,
		HostKeys:                config.SSH.ServerHostKeys,
		TrustedUserCAKeys:       config.SSH.TrustedUserCAKeys,
		TrustedUserCAKeysParsed: config.SSH.TrustedUserCAKeysParsed,
		KeepAliveInterval:       config.SSH.KeepAliveInterval,
		Verifier:                vierifier,
		RepoCtrl:                repoctrl,
	}
}
