// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package runner

import (
	"strings"

	"github.com/drone/drone-runtime/engine"
	"github.com/drone/drone-runtime/runtime"
	"github.com/drone/drone/core"
)

func convertVolumes(from []string) map[string]string {
	to := map[string]string{}
	for _, s := range from {
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := parts[1]
		to[key] = val
	}
	return to
}

func convertSecrets(from []*core.Secret) map[string]string {
	to := map[string]string{}
	for _, secret := range from {
		to[secret.Name] = secret.Data
	}
	return to
}

func convertRegistry(from []*core.Registry) []*engine.DockerAuth {
	var to []*engine.DockerAuth
	for _, registry := range from {
		to = append(to, &engine.DockerAuth{
			Address:  registry.Address,
			Username: registry.Username,
			Password: registry.Password,
		})
	}
	return to
}

func convertLines(from []*runtime.Line) []*core.Line {
	var to []*core.Line
	for _, v := range from {
		to = append(to, &core.Line{
			Number:    v.Number,
			Message:   v.Message,
			Timestamp: v.Timestamp,
		})
	}
	return to
}

func convertLine(from *runtime.Line) *core.Line {
	return &core.Line{
		Number:    from.Number,
		Message:   from.Message,
		Timestamp: from.Timestamp,
	}
}
