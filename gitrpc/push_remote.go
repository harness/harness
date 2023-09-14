// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"
	"github.com/harness/gitness/gitrpc/rpc"
)

type PushRemoteParams struct {
	ReadParams
	RemoteUrlWithToken string
	Timeout            int64
}

func (c *Client) PushRemote(ctx context.Context, params *PushRemoteParams) error {
	if params == nil {
		return ErrNoParamsProvided
	}

	_, err := c.pushService.PushRemote(ctx, &rpc.PushRemoteRequest{
		Base:               mapToRPCReadRequest(params.ReadParams),
		RemoteUrlWithToken: params.RemoteUrlWithToken,
		Timeout:            params.Timeout,
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

	if p.RemoteUrlWithToken == "" {
		return ErrInvalidArgumentf("remote url cannot be empty")
	}
	return nil
}
