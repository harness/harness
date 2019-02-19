// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package queue

import (
	"context"
	"errors"

	"github.com/drone/drone/core"
)

type scheduler struct {
	*queue
	*canceller
}

// New creates a new scheduler.
func New(store core.StageStore) core.Scheduler {
	return &scheduler{
		queue:     newQueue(store),
		canceller: newCanceller(),
	}
}

func (d *scheduler) Stats(context.Context) (interface{}, error) {
	return nil, errors.New("not implemented")
}
