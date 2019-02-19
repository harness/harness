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

// func Test_SecretManager_List_SkipDockerAuthConfig(t *testing.T) {
// 	manager := secretManager{
// 		repo: &core.Repository{
// 			Secret: "m5bahAG7YVp114R4YgMv5uW7bTEzx7yn",
// 		},
// 		build: &core.Build{
// 			Event: core.EventPush,
// 		},
// 		config: &yaml.Manifest{
// 			Resources: []yaml.Resource{
// 				&yaml.Secret{
// 					Kind: "secret",
// 					Type: "encrypted",
// 					Data: map[string]string{
// 						"DOCKER_AUTH_CONFIG": "LiDvQo6Zw5ArpwCByD4Pb9DAibl5bMaUInzXFT93sEoejT_jNZQCtXpIbuGJh7Iw3ixyd8vMDC0vXiQWw5VhKvLWLKg=",
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
// 	if len(got) != 0 {
// 		t.Errorf("Expect DOCKER_AUTH_CONFIG excluded from secret list")
// 	}
// }

// func Test_SecretManager_ListInline(t *testing.T) {
// 	manager := secretManager{
// 		repo: &core.Repository{
// 			Secret: "dvBIW3c7P5WW0iwMaPNKRCKIN19NgqMH",
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
// 						"docker_password": "5OXQwLXkLY0eWcqx0oM7SzY6nKrMBBUlRIC5aod0kmRH0-85AaH-4itxTrS21VaG88NESE5HB5Klq9QtTkAXsaW9KQ==",
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
// 	want := []*core.Secret{
// 		{
// 			Name: "docker_password",
// 			Data: "correct-horse-battery-staple",
// 		},
// 	}
// 	if diff := cmp.Diff(got, want); diff != "" {
// 		t.Errorf(diff)
// 	}
// }

// func Test_SecretManager_ListInline_SkipPull(t *testing.T) {
// 	manager := secretManager{
// 		repo: &core.Repository{
// 			Secret: "dvBIW3c7P5WW0iwMaPNKRCKIN19NgqMH",
// 			Slug:   "octocat/hello-world",
// 		},
// 		build: &core.Build{
// 			Event: core.EventPullRequest,
// 			Fork:  "octocat/hello-world",
// 		},
// 		config: &yaml.Manifest{
// 			Resources: []yaml.Resource{
// 				&yaml.Secret{
// 					Kind: "secret",
// 					Type: "encrypted",
// 					Data: map[string]string{
// 						"docker_password": "5OXQwLXkLY0eWcqx0oM7SzY6nKrMBBUlRIC5aod0kmRH0-85AaH-4itxTrS21VaG88NESE5HB5Klq9QtTkAXsaW9KQ==",
// 					},
// 				},
// 			},
// 		},
// 	}
// 	secrets, err := manager.list(noContext)
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}
// 	if len(secrets) != 0 {
// 		t.Errorf("Expect secret not exposed to a pull request")
// 	}
// }

// func Test_SecretManager_ListInline_DecryptErr(t *testing.T) {
// 	manager := secretManager{
// 		repo: &core.Repository{
// 			Secret: "invalid",
// 		},
// 		build: &core.Build{
// 			Event: core.EventPush,
// 		},
// 		config: &yaml.Manifest{
// 			Resources: []yaml.Resource{
// 				&yaml.Secret{
// 					Kind: "secret",
// 					Type: "encrypted",
// 					Data: map[string]string{
// 						"docker_password": "LiDvQo6Zw5ArpwCByD4Pb9DAibl5bMaUInzXFT93sEoejT_jNZQCtXpIbuGJh7Iw3ixyd8vMDC0vXiQWw5VhKvLWLKg=",
// 					},
// 				},
// 			},
// 		},
// 	}
// 	_, err := manager.list(noContext)
// 	if err == nil {
// 		t.Errorf("Expect decryption error")
// 	}
// }

// func Test_SecretManager_ListInline_DecodeErr(t *testing.T) {
// 	manager := secretManager{
// 		repo: &core.Repository{
// 			Secret: "m5bahAG7YVp114R4YgMv5uW7bTEzx7yn",
// 		},
// 		build: &core.Build{
// 			Event: core.EventPush,
// 		},
// 		config: &yaml.Manifest{
// 			Resources: []yaml.Resource{
// 				&yaml.Secret{
// 					Kind: "secret",
// 					Type: "encrypted",
// 					Data: map[string]string{
// 						"docker_password": "nNOfLyHNFMecBwWq4DxGIkIRqfCX3DElxc7sejue",
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

// func Test_SecretManager_ListExternal(t *testing.T) {
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
// 		Name: "docker_password",
// 		Data: "correct-horse-battery-staple",
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

// 	service := mock.NewMockSecretService(controller)
// 	service.EXPECT().Find(gomock.Any(), gomock.Any()).DoAndReturn(mockResp)

// 	manager := secretManager{
// 		repo:  mockRepo,
// 		build: mockBuild,
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
// 		remote: service,
// 	}
// 	got, err := manager.list(noContext)
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}
// 	want := []*core.Secret{
// 		{
// 			Name: "docker_password",
// 			Data: "correct-horse-battery-staple",
// 		},
// 	}
// 	if diff := cmp.Diff(got, want); diff != "" {
// 		t.Errorf(diff)
// 	}
// }

// func Test_SecretManager_ListExternal_Err(t *testing.T) {
// 	controller := gomock.NewController(t)
// 	defer controller.Finish()

// 	service := mock.NewMockSecretService(controller)
// 	service.EXPECT().Find(gomock.Any(), gomock.Any()).Return(nil, io.EOF)

// 	manager := secretManager{
// 		repo: &core.Repository{
// 			Slug: "octocat/hello-world",
// 		},
// 		build: &core.Build{
// 			Event: core.EventPush,
// 		},
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
// 		remote: service,
// 	}
// 	_, err := manager.list(noContext)
// 	if err == nil {
// 		t.Errorf("Expect error fetching external secret")
// 	}
// }

// // func Test_extractCiphertext(t *testing.T) {
// // 	tests := []struct {
// // 		secret     config.Secret
// // 		algorithm  string
// // 		ciphertext string
// // 		ok         bool
// // 	}{
// // 		{
// // 			secret:     config.Secret{Secretbox: "LiDvQo6Zw5ArpwCByD4Pb9DAibl5bMaUInzXFT93sEoejT_jNZQCtXpIbuGJh7Iw3ixyd8vMDC0vXiQWw5VhKvLWLKg="},
// // 			algorithm:  core.EncryptSecretBox,
// // 			ciphertext: "LiDvQo6Zw5ArpwCByD4Pb9DAibl5bMaUInzXFT93sEoejT_jNZQCtXpIbuGJh7Iw3ixyd8vMDC0vXiQWw5VhKvLWLKg=",
// // 			ok:         true,
// // 		},
// // 		{
// // 			secret:     config.Secret{Aesgcm: "JjnUFKmN-H0GJmXO8oByrgZoCb0imNTcGgV496TNB7Y3MESCerxYvxjWWP1RQdPibfT1P97F1WA="},
// // 			algorithm:  core.EncryptAESGCM,
// // 			ciphertext: "JjnUFKmN-H0GJmXO8oByrgZoCb0imNTcGgV496TNB7Y3MESCerxYvxjWWP1RQdPibfT1P97F1WA=",
// // 			ok:         true,
// // 		},
// // 		{
// // 			secret: config.Secret{},
// // 			ok:     false,
// // 		},
// // 	}
// // 	for i, test := range tests {
// // 		algorithm, ciphertext, ok := extractCiphertext(test.secret)
// // 		if got, want := algorithm, test.algorithm; got != want {
// // 			t.Errorf("Want algorithm %s at index %v", want, i)
// // 		}
// // 		if got, want := ciphertext, test.ciphertext; got != want {
// // 			t.Errorf("Want ciphertext %s at index %v", want, i)
// // 		}
// // 		if got, want := ok, test.ok; got != want {
// // 			t.Errorf("Want ok %v at index %v", want, i)
// // 		}
// // 	}
// // }

// func Test_decrypt(t *testing.T) {
// 	tests := []struct {
// 		Key        string
// 		Algorithm  string
// 		Ciphertext string
// 		Plaintext  string
// 	}{
// 		{
// 			Algorithm:  core.EncryptSecretBox,
// 			Plaintext:  "correct-horse-battery-staple",
// 			Ciphertext: "LiDvQo6Zw5ArpwCByD4Pb9DAibl5bMaUInzXFT93sEoejT_jNZQCtXpIbuGJh7Iw3ixyd8vMDC0vXiQWw5VhKvLWLKg=",
// 			Key:        "m5bahAG7YVp114R4YgMv5uW7bTEzx7yn",
// 		},
// 		{
// 			Algorithm:  core.EncryptAESGCM,
// 			Plaintext:  "correct-horse-battery-staple",
// 			Ciphertext: "JjnUFKmN-H0GJmXO8oByrgZoCb0imNTcGgV496TNB7Y3MESCerxYvxjWWP1RQdPibfT1P97F1WA=",
// 			Key:        "m5bahAG7YVp114R4YgMv5uW7bTEzx7yn",
// 		},
// 	}
// 	for i, test := range tests {
// 		plaintext, _ := decrypt(test.Algorithm, test.Ciphertext, test.Key)
// 		if got, want := plaintext, test.Plaintext; got != want {
// 			t.Errorf("Want %v at index %v", want, i)
// 		}
// 	}
// }

// func Test_isDockerConfig(t *testing.T) {
// 	tests := []struct {
// 		Name  string
// 		Match bool
// 	}{
// 		{
// 			Name:  "docker_auth_config",
// 			Match: true,
// 		},
// 		{
// 			Name:  "DOCKER_auth_CONFIG",
// 			Match: true,
// 		},
// 		{
// 			Name:  "docker_config",
// 			Match: false,
// 		},
// 	}
// 	for i, test := range tests {
// 		if got, want := isDockerConfig(test.Name), test.Match; got != want {
// 			t.Errorf("Want %v at index %v", want, i)
// 		}
// 	}
// }
