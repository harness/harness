// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package runner

import "github.com/drone/drone/core"

func toSecretMap(secrets []*core.Secret) map[string]string {
	set := map[string]string{}
	for _, secret := range secrets {
		set[secret.Name] = secret.Data
	}
	return set
}

// import (
// 	"context"
// 	"encoding/json"
// 	"strings"

// 	"github.com/drone/drone-yaml/yaml"
// 	"github.com/drone/drone/core"
// 	"github.com/drone/drone/crypto/aesgcm"
// 	"github.com/drone/drone/crypto/secretbox"
// )

// var noContext = context.Background()

// type secretManager struct {
// 	repo   *core.Repository
// 	build  *core.Build
// 	config *yaml.Manifest
// 	remote core.SecretService
// }

// func (s *secretManager) list(_ context.Context) ([]*core.Secret, error) {
// 	var secrets []*core.Secret
// 	for _, resource := range s.config.Resources {
// 		res, ok := resource.(*yaml.Secret)
// 		if !ok {
// 			continue
// 		}
// 		for name, value := range res.Data {
// 			// skip secrets the are intended for use with authenticating
// 			// to the docker registry and pulling private images.
// 			if isDockerConfig(name) {
// 				continue
// 			}

// 			if res.Type == "encrypted" {
// 				value = strings.Replace(value, " ", "", -1)
// 				value = strings.Replace(value, "\n", "", -1)

// 				plaintext, err := decrypt(core.EncryptAESGCM, value, s.repo.Secret)
// 				if err != nil {
// 					return nil, err
// 				}
// 				secret := new(core.Secret)
// 				secret.Name = name
// 				err = json.Unmarshal([]byte(plaintext), secret)
// 				if err != nil {
// 					return nil, err
// 				}
// 				if secret.Pull == false && s.build.Event == core.EventPullRequest {
// 					continue
// 				}
// 				secrets = append(secrets, secret)
// 			} else {
// 				// the user has the option of aliasing the
// 				// secret name. If the user specifies an external
// 				// name it must be used for the external query.
// 				req := &core.SecretRequest{
// 					Name:  value,
// 					Repo:  s.repo,
// 					Build: s.build,
// 				}

// 				// if s.repo.Endpoints.Secret.Endpoint != "" {
// 				// 	// fetch the secret from the user-defined endpoint.
// 				// 	secret, err := s.remote.FindEndpoint(noContext, req, s.repo.Endpoints.Secret)
// 				// 	if err != nil {
// 				// 		return nil, err
// 				// 	}
// 				// 	if secret == nil {
// 				// 		continue
// 				// 	}
// 				// 	secrets = append(secrets, &core.Secret{
// 				// 		Name: name, // use the aliased name.
// 				// 		Data: secret.Data,
// 				// 	})
// 				// } else {
// 				// fetch the secret from the global endpoint.
// 				secret, err := s.remote.Find(noContext, req)
// 				if err != nil {
// 					return nil, err
// 				}
// 				if secret == nil {
// 					continue
// 				}
// 				secrets = append(secrets, &core.Secret{
// 					Name: name, // use the aliased name.
// 					Data: secret.Data,
// 				})
// 				// }
// 			}
// 		}
// 	}
// 	return secrets, nil
// }

// // helper function extracts the ciphertext and algorithm type
// // // from the yaml secret structure.
// // func extractCiphertext(secret yaml.Secret) (algorithm, ciphertext string, ok bool) {
// // 	return core.EncryptAESGCM, secret.Data, true
// // }

// // helper funciton decrypts the ciphertext using the provided
// // decryption algorithm and decryption key.
// func decrypt(algorithm, ciphertext, key string) (string, error) {
// 	switch algorithm {
// 	case core.EncryptAESGCM:
// 		return aesgcm.DecryptString(ciphertext, key)
// 	default:
// 		return secretbox.Decrypt(ciphertext, key)
// 	}
// }

// // helper function returns true if the build event matches the
// // docker_auth_config variable name.
// func isDockerConfig(name string) bool {
// 	return strings.EqualFold(name, "DOCKER_AUTH_CONFIG")
// }
