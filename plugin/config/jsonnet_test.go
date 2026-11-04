// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

//go:build !oss
// +build !oss

package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"
	"github.com/golang/mock/gomock"
)

func TestJsonnetRejectsServerFileImports(t *testing.T) {
	file := filepath.Join(t.TempDir(), "server-secret.json")
	if err := os.WriteFile(file, []byte(`{"secret":"server-file-canary"}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, syntax := range []string{"import", "importstr", "importbin"} {
		t.Run(syntax, func(t *testing.T) {
			controller := gomock.NewController(t)
			files := mock.NewMockFileService(controller)
			files.EXPECT().Find(gomock.Any(), gomock.Any(), "owner/repo", "commit", "", ".drone.jsonnet").Return(
				&core.File{Data: []byte(fmt.Sprintf("[{ secret: %s %q }]", syntax, filepath.ToSlash(file)))}, nil,
			)
			service := Jsonnet(files, true)
			_, err := service.Find(context.Background(), &core.ConfigArgs{
				Repo:  &core.Repository{Slug: "owner/repo", Config: ".drone.jsonnet"},
				Build: &core.Build{After: "commit"},
			})
			if err == nil {
				t.Fatal("server file import succeeded")
			}
		})
	}
}
