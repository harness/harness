// Copyright 2021 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redisdb

import (
	"context"
)

// LockErr is an interface with lock and unlock functions that return an error.
// Method names are chosen so that redsync.Mutex implements the interface.
type LockErr interface {
	LockContext(context.Context) error
	UnlockContext(context.Context) (bool, error)
}

// LockErrNoOp is a dummy no-op locker
type LockErrNoOp struct{}

func (l LockErrNoOp) LockContext(context.Context) error           { return nil }
func (l LockErrNoOp) UnlockContext(context.Context) (bool, error) { return false, nil }
