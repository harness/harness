// Copyright 2019 Drone IO, Inc.
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

package sink

import (
	"context"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"
	"github.com/drone/drone/version"
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
	users.EXPECT().ListRange(gomock.Any(), gomock.Any()).Return([]*core.User{{Email: "jane@acme.com", Created: 1257894000}}, nil)

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
	d.config.License = "trial"
	d.config.EnableGithub = true
	d.config.EnableAgents = true
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
			"host": "test.example.com",
			"tags": ["version:` + version.Version.String() + `","remote:github:cloud","scheduler:internal:agents","license:trial","installer:jane@acme.com","installed:2009-11-10T23:00:00Z"]
		},
		{
			"metric": "drone.repos",
			"points": [[915148800, 20]],
			"type": "gauge",
			"host": "test.example.com",
			"tags": ["version:` + version.Version.String() + `","remote:github:cloud","scheduler:internal:agents","license:trial"]
		},
		{
			"metric": "drone.builds",
			"points": [[915148800, 30]],
			"type": "gauge",
			"host": "test.example.com",
			"tags": ["version:` + version.Version.String() + `","remote:github:cloud","scheduler:internal:agents","license:trial"]
		}
    ]
}`
