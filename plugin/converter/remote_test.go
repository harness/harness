// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package converter

import (
	"context"
	"testing"
	"time"

	"github.com/drone/drone/core"
	"github.com/h2non/gock"
)

func TestConvert(t *testing.T) {
	defer gock.Off()

	gock.New("https://company.com").
		Post("/convert").
		MatchHeader("Accept", "application/vnd.drone.convert.v1\\+json").
		MatchHeader("Accept-Encoding", "identity").
		MatchHeader("Content-Type", "application/json").
		Reply(200).
		BodyString(`{"data": "{ kind: pipeline, type: docker, name: default }"}`).
		Done()

	args := &core.ConvertArgs{
		User:  &core.User{Login: "octocat"},
		Repo:  &core.Repository{Slug: "octocat/hello-world", Config: ".drone.yml"},
		Build: &core.Build{After: "6d144de7"},
		Config: &core.Config{
			Data: "{ kind: pipeline, name: default }",
		},
	}

	service := Remote("https://company.com/convert", "GMEuUHQfmrMRsseWxi9YlIeBtn9lm6im", "",
		false, time.Minute)
	result, err := service.Convert(context.Background(), args)
	if err != nil {
		t.Error(err)
		return
	}

	if result.Data != "{ kind: pipeline, type: docker, name: default }" {
		t.Errorf("unexpected file contents")
	}

	if gock.IsPending() {
		t.Errorf("Unfinished requests")
		return
	}
}
