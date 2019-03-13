// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package rpc

import (
	"sync"

	"github.com/drone/drone/core"
	"github.com/drone/drone/operator/manager"
)

type requestRequest struct {
	Request *manager.Request
}

type acceptRequest struct {
	Stage   int64
	Machine string
}

type netrcRequest struct {
	Repo int64
}

type detailsRequest struct {
	Stage int64
}

type stageRequest struct {
	Stage *core.Stage
}

type stepRequest struct {
	Step *core.Step
}

type writeRequest struct {
	Step int64
	Line *core.Line
}

type watchRequest struct {
	Build int64
}

type watchResponse struct {
	Done bool
}

type buildContextToken struct {
	Secret  string
	Context *manager.Context
}

type errorWrapper struct {
	Message string
}

var writePool = sync.Pool{
	New: func() interface{} {
		return &writeRequest{}
	},
}
