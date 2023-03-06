// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"

	"github.com/harness/gitness/gitrpc/enum"
	"github.com/harness/gitness/gitrpc/rpc"
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
	refType := enum.RefToRPC(params.Type)
	if refType == rpc.RefType_Undefined {
		return GetRefResponse{}, ErrInvalidArgumentf("invalid argument: '%s'", refType)
	}

	result, err := c.refService.GetRef(ctx, &rpc.GetRefRequest{
		Base:    mapToRPCReadRequest(params.ReadParams),
		RefName: params.Name,
		RefType: refType,
	})
	if err != nil {
		return GetRefResponse{}, processRPCErrorf(err, "failed to get %s ref '%s'", params.Type.String(), params.Name)
	}

	return GetRefResponse{SHA: result.Sha}, nil
}

type UpdateRefParams struct {
	WriteParams
	Type enum.RefType
	Name string
	// NewValue specified the new value the reference should point at.
	// An empty value will lead to the deletion of the branch.
	NewValue string
	// OldValue is an optional value that can be used to ensure that the reference
	// is updated iff its current value is matching the provided value.
	OldValue string
}

func (c *Client) UpdateRef(ctx context.Context, params UpdateRefParams) error {
	refType := enum.RefToRPC(params.Type)
	if refType == rpc.RefType_Undefined {
		return ErrInvalidArgumentf("invalid argument: '%s'", refType)
	}

	_, err := c.refService.UpdateRef(ctx, &rpc.UpdateRefRequest{
		Base:     mapToRPCWriteRequest(params.WriteParams),
		RefName:  params.Name,
		RefType:  refType,
		NewValue: params.NewValue,
		OldValue: params.OldValue,
	})
	if err != nil {
		return processRPCErrorf(err, "failed to update %s ref '%s'", params.Type.String(), params.Name)
	}

	return err
}
