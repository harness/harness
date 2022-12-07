// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import "github.com/harness/gitness/types"

type PullReq struct {
	types.PullReq
	Author types.Identity  `json:"author"`
	Merger *types.Identity `json:"merger"`
}

func mapPullReqInfo(pri *types.PullReqInfo) *PullReq {
	pr := &PullReq{}
	pr.PullReq = pri.PullReq
	pr.Author = types.Identity{
		ID:    pri.AuthorID,
		Name:  pri.AuthorName,
		Email: pri.AuthorEmail,
	}
	if pr.MergedBy != nil {
		pr.Merger = &types.Identity{
			ID:    *pri.MergerID,
			Name:  *pri.MergerName,
			Email: *pri.MergerEmail,
		}
	}
	return pr
}
