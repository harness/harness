// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"

	"github.com/harness/gitness/gitrpc/rpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RefType int

const (
	RefTypeBranch RefType = iota
	RefTypeTag
)

type GetRefParams struct {
	ReadParams
	Name string
	Type RefType
}

type GetRefResponse struct {
	SHA string
}

func (c *Client) GetRef(ctx context.Context, params *GetRefParams) (*GetRefResponse, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	var refType rpc.GetRefRequest_RefType
	switch params.Type {
	case RefTypeBranch:
		refType = rpc.GetRefRequest_Branch
	case RefTypeTag:
		refType = rpc.GetRefRequest_Tag
	default:
		return nil, ErrInvalidArgument
	}

	result, err := c.refService.GetRef(ctx, &rpc.GetRefRequest{
		Base:    mapToRPCReadRequest(params.ReadParams),
		RefName: params.Name,
		RefType: refType,
	})
	if s, ok := status.FromError(err); err != nil && ok && s.Code() == codes.NotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &GetRefResponse{
		SHA: result.Sha,
	}, nil
}
