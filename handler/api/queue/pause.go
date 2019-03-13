// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package queue

// import (
// 	"net/http"

// 	"github.com/drone/drone/core"
// 	"github.com/drone/drone/handler/api/render"
// 	"github.com/drone/drone/logger"
// )

// // HandlePause returns an http.HandlerFunc that processes
// // an http.Request to pause the queue.
// func HandlePause(queue core.Queue) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		ctx := r.Context()
// 		err := queue.Pause(ctx)
// 		if err != nil {
// 			render.InternalError(w, err)
// 			logger.FromRequest(r).WithError(err).
// 				Errorln("api: cannot pause queue")
// 			return
// 		}
// 		w.WriteHeader(http.StatusNoContent)
// 	}
// }
