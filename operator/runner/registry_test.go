// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package runner

// import (
// 	"context"
// 	"encoding/json"
// 	"io"
// 	"testing"

// 	"github.com/drone/drone-yaml/yaml"
// 	"github.com/drone/drone/core"
// 	"github.com/drone/drone/mock"

// 	"github.com/golang/mock/gomock"
// 	"github.com/google/go-cmp/cmp"
// )

// func Test_RegistryManager_ListExternal(t *testing.T) {
// 	controller := gomock.NewController(t)
// 	defer controller.Finish()

// 	want := []*core.Registry{
// 		{
// 			Address:  "docker.io",
// 			Username: "octocat",
// 			Password: "pa55word",
// 		},
// 	}

// 	service := mock.NewMockRegistryService(controller)
// 	service.EXPECT().List(gomock.Any(), gomock.Any()).Return(want, nil)
// 	service.EXPECT().ListEndpoint(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

// 	manager := registryManager{
// 		auths:  service,
// 		config: &yaml.Manifest{},
// 		repo:   &core.Repository{},
// 	}
// 	got, err := manager.list(noContext)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	if diff := cmp.Diff(got, want); diff != "" {
// 		t.Errorf(diff)
// 	}
// }

// // this test verifies that the registry credential manager
// // exits and returns an error if unable to fetch registry
// // credentials from the external provider.
// func Test_RegistryManager_ListExternal_Err(t *testing.T) {
// 	controller := gomock.NewController(t)
// 	defer controller.Finish()

// 	service := mock.NewMockRegistryService(controller)
// 	service.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, io.EOF)

// 	manager := registryManager{
// 		auths: service,
// 	}
// 	_, err := manager.list(noContext)
// 	if err == nil {
// 		t.Errorf("Expect error fetching external secret")
// 	}
// }

// // this test verifies that the registry credential manager
// // skips secrets that are not docker_auth_config files.
// func Test_RegistryManager_ListInternal_Skip(t *testing.T) {
// 	controller := gomock.NewController(t)
// 	defer controller.Finish()

// 	service := mock.NewMockRegistryService(controller)
// 	service.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil)
// 	service.EXPECT().ListEndpoint(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

// 	manager := registryManager{
// 		repo:  &core.Repository{},
// 		auths: service,
// 		config: &yaml.Manifest{
// 			Resources: []yaml.Resource{
// 				&yaml.Secret{
// 					Kind: "secret",
// 					Type: "external",
// 					Data: map[string]string{
// 						"docker_password": "docker_password",
// 					},
// 				},
// 			},
// 		},
// 	}

// 	got, err := manager.list(noContext)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	var want []*core.Registry
// 	if diff := cmp.Diff(got, want); diff != "" {
// 		t.Errorf(diff)
// 	}
// }

// // this test verifies that the registry credential manager
// // fetches registry credentials from the remote secret store,
// // and successfully parses the .docker/config.json contents.
// func Test_RegistryManager_ListExternalSecrets(t *testing.T) {
// 	controller := gomock.NewController(t)
// 	defer controller.Finish()

// 	mockRepo := &core.Repository{
// 		Slug: "octocat/hello-world",
// 	}

// 	mockBuild := &core.Build{
// 		Event: core.EventPullRequest,
// 		Fork:  "octocat/hello-world",
// 	}

// 	mockSecret := &core.Secret{
// 		Name: "docker_auth_config",
// 		Data: `{"auths": {"index.docker.io": {"auth": "b2N0b2NhdDpjb3JyZWN0LWhvcnNlLWJhdHRlcnktc3RhcGxl"}}}`,
// 	}

// 	mockSecretReq := &core.SecretRequest{
// 		Name:  mockSecret.Name,
// 		Repo:  mockRepo,
// 		Build: mockBuild,
// 	}

// 	mockResp := func(ctx context.Context, req *core.SecretRequest) (*core.Secret, error) {
// 		if diff := cmp.Diff(req, mockSecretReq); diff != "" {
// 			t.Errorf(diff)
// 		}
// 		return mockSecret, nil
// 	}

// 	registries := mock.NewMockRegistryService(controller)
// 	registries.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil)
// 	registries.EXPECT().ListEndpoint(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

// 	secrets := mock.NewMockSecretService(controller)
// 	secrets.EXPECT().Find(gomock.Any(), gomock.Any()).DoAndReturn(mockResp)

// 	manager := registryManager{
// 		auths:   registries,
// 		secrets: secrets,
// 		repo:    mockRepo,
// 		build:   mockBuild,
// 		config: &yaml.Manifest{
// 			Resources: []yaml.Resource{
// 				&yaml.Secret{
// 					Kind: "secret",
// 					Type: "external",
// 					Data: map[string]string{
// 						"docker_auth_config": "docker_auth_config",
// 					},
// 				},
// 			},
// 		},
// 	}
// 	want := []*core.Registry{
// 		{
// 			Address:  "index.docker.io",
// 			Username: "octocat",
// 			Password: "correct-horse-battery-staple",
// 		},
// 	}
// 	got, err := manager.list(noContext)
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}
// 	if diff := cmp.Diff(got, want); diff != "" {
// 		t.Errorf(diff)
// 	}
// }

// // this test verifies that the registry credential manager
// // fetches registry credentials from the remote secret store,
// // and returns an error if external rpc call fails.
// func Test_RegistryManager_ListExternalSecrets_Err(t *testing.T) {
// 	controller := gomock.NewController(t)
// 	defer controller.Finish()

// 	registries := mock.NewMockRegistryService(controller)
// 	registries.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil)
// 	registries.EXPECT().ListEndpoint(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

// 	secrets := mock.NewMockSecretService(controller)
// 	secrets.EXPECT().Find(gomock.Any(), gomock.Any()).Return(nil, io.EOF)

// 	manager := registryManager{
// 		repo:    &core.Repository{},
// 		auths:   registries,
// 		secrets: secrets,
// 		config: &yaml.Manifest{
// 			Resources: []yaml.Resource{
// 				&yaml.Secret{
// 					Kind: "secret",
// 					Type: "external",
// 					Data: map[string]string{
// 						"docker_auth_config": "docker_auth_config",
// 					},
// 				},
// 			},
// 		},
// 	}

// 	_, err := manager.list(noContext)
// 	if err == nil {
// 		t.Errorf("Expect error")
// 	}
// }

// // this test verifies that the registry credential manager
// // fetches registry credentials from the remote secret store,
// // and returns an error if the .docker/config.json contents
// // cannot be unmarshaled.
// func Test_RegistryManager_ListExternalSecrets_ParseErr(t *testing.T) {
// 	controller := gomock.NewController(t)
// 	defer controller.Finish()

// 	mockSecret := &core.Secret{
// 		Name: "docker_auth_config",
// 		Data: `[]`,
// 	}

// 	registries := mock.NewMockRegistryService(controller)
// 	registries.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil)
// 	registries.EXPECT().ListEndpoint(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

// 	secrets := mock.NewMockSecretService(controller)
// 	secrets.EXPECT().Find(gomock.Any(), gomock.Any()).Return(mockSecret, nil)

// 	manager := registryManager{
// 		auths:   registries,
// 		secrets: secrets,
// 		repo: &core.Repository{
// 			Slug: "octocat/hello-world",
// 		},
// 		build: &core.Build{
// 			Event: core.EventPush,
// 			Fork:  "octocat/hello-world",
// 		},
// 		config: &yaml.Manifest{
// 			Resources: []yaml.Resource{
// 				&yaml.Secret{
// 					Kind: "secret",
// 					Type: "external",
// 					Data: map[string]string{
// 						"docker_auth_config": "docker_auth_config",
// 					},
// 				},
// 			},
// 		},
// 	}

// 	_, err := manager.list(noContext)
// 	if _, ok := err.(*json.UnmarshalTypeError); !ok {
// 		t.Errorf("Expect decoding error")
// 	}
// }

// // this test verifies that the registry credential manager
// // can decrypt inline registry credentials included in the yaml,
// // where the encrypted content is a .docker/config.json file.
// func Test_RegistryManager_ListInline(t *testing.T) {
// 	controller := gomock.NewController(t)
// 	defer controller.Finish()

// 	if true {
// 		t.Skipf("skip docker_auth_config encryption test")
// 		return
// 	}

// 	registries := mock.NewMockRegistryService(controller)
// 	registries.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil)
// 	registries.EXPECT().ListEndpoint(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

// 	manager := registryManager{
// 		auths: registries,
// 		repo: &core.Repository{
// 			Secret: "m5bahAG7YVp114R4YgMv5uW7bTEzx7yn",
// 			Slug:   "octocat/hello-world",
// 		},
// 		build: &core.Build{
// 			Event: core.EventPush,
// 			Fork:  "octocat/hello-world",
// 		},
// 		config: &yaml.Manifest{
// 			Resources: []yaml.Resource{
// 				&yaml.Secret{
// 					Kind: "secret",
// 					Type: "encrypted",
// 					Data: map[string]string{
// 						"docker_auth_config": "0jye_JUWxgu1qZRd56d9GSnl3-gJgsBAakeKAQ4BX_UDSvT0ntcwXT38KfiI5OY-BNZSKwfoQrQuPYn2VJWXcUMSmy0JLdBEDzWJ-m8s-KPBApuh6vVTafKzrslK-E0P7ZfqiR0ulXWsHqJhzVXInjITx8oxsmcZ458Fwbvk6gXLudRsKKr6RjI4Jcr4mQGT",
// 					},
// 				},
// 			},
// 		},
// 	}

// 	got, err := manager.list(noContext)
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}

// 	want := []*core.Registry{
// 		{
// 			Address:  "index.docker.io",
// 			Username: "octocat",
// 			Password: "correct-horse-battery-staple",
// 		},
// 	}
// 	if diff := cmp.Diff(got, want); diff != "" {
// 		t.Errorf(diff)
// 	}
// }
