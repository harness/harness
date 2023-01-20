// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"

	"github.com/harness/gitness/gitrpc/enum"
	"github.com/harness/gitness/gitrpc/rpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetRefParams struct {
	ReadParams
	Name string
	Type enum.RefType
}

type GetRefResponse struct {
	SHA string
}

func (c *Client) GetRef(ctx context.Context, params GetRefParams) (GetRefResponse, error) {
	refType, isOk := enum.RefToRPC(params.Type)
	if !isOk {
		return GetRefResponse{}, ErrInvalidArgument
	}

	result, err := c.refService.GetRef(ctx, &rpc.GetRefRequest{
		Base:    mapToRPCReadRequest(params.ReadParams),
		RefName: params.Name,
		RefType: refType,
	})
	if s, ok := status.FromError(err); err != nil && ok && s.Code() == codes.NotFound {
		return GetRefResponse{}, ErrNotFound
	}

	return GetRefResponse{SHA: result.Sha}, nil
}

type UpdateRefParams struct {
	WriteParams
	Name     string
	Type     enum.RefType
	NewValue string
	OldValue string
}

func (c *Client) UpdateRef(ctx context.Context, params UpdateRefParams) error {
	refType, isOk := enum.RefToRPC(params.Type)
	if !isOk {
		return ErrInvalidArgument
	}

	_, err := c.refService.UpdateRef(ctx, &rpc.UpdateRefRequest{
		Base:     mapToRPCWriteRequest(params.WriteParams),
		RefName:  params.Name,
		RefType:  refType,
		NewValue: params.NewValue,
		OldValue: params.OldValue,
	})
	if s, ok := status.FromError(err); err != nil && ok && s.Code() == codes.NotFound {
		return ErrNotFound
	}

	return err
}
