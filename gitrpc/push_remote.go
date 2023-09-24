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

package gitrpc

import (
	"context"

	"github.com/harness/gitness/gitrpc/rpc"
)

type PushRemoteParams struct {
	ReadParams
	RemoteUrl string
}

func (c *Client) PushRemote(ctx context.Context, params *PushRemoteParams) error {
	if params == nil {
		return ErrNoParamsProvided
	}

	_, err := c.pushService.PushRemote(ctx, &rpc.PushRemoteRequest{
		Base:      mapToRPCReadRequest(params.ReadParams),
		RemoteUrl: params.RemoteUrl,
	})
	if err != nil {
		return processRPCErrorf(err, "failed to push to remote")
	}

	return nil
}

func (p PushRemoteParams) Validate() error {
	if err := p.ReadParams.Validate(); err != nil {
		return err
	}

	if p.RemoteUrl == "" {
		return ErrInvalidArgumentf("remote url cannot be empty")
	}
	return nil
}
