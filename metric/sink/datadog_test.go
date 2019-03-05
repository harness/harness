// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package sink

import (
	"context"
	"testing"

	"github.com/drone/drone/mock"
	"github.com/golang/mock/gomock"
	"github.com/h2non/gock"
)

var noContext = context.Background()

func TestDo(t *testing.T) {
	controller := gomock.NewController(t)

	gock.InterceptClient(httpClient)
	defer func() {
		gock.RestoreClient(httpClient)
		gock.Off()
		controller.Finish()
	}()

	users := mock.NewMockUserStore(controller)
	users.EXPECT().Count(gomock.Any()).Return(int64(10), nil)

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().Count(gomock.Any()).Return(int64(20), nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().Count(gomock.Any()).Return(int64(30), nil)

	gock.New("https://api.datadoghq.com").
		Post("/api/v1/series").
		JSON(sample).
		Reply(200)

	d := new(Datadog)
	d.users = users
	d.repos = repos
	d.builds = builds
	d.system.Host = "test.example.com"
	d.config.Endpoint = "https://api.datadoghq.com/api/v1/series"
	d.do(noContext, 915148800)

	if gock.IsPending() {
		t.Errorf("Unfinished requests")
	}
}

var sample = `{
	"series" : [
		{
			"metric": "drone.users",
			"points": [[915148800, 10]],
			"type": "gauge",
			"host": "test.example.com"
		},
		{
			"metric": "drone.repos",
			"points": [[915148800, 20]],
			"type": "gauge",
			"host": "test.example.com"
		},
		{
			"metric": "drone.builds",
			"points": [[915148800, 30]],
			"type": "gauge",
			"host": "test.example.com"
		}
    ]
}`
