// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package runner

// import (
// 	"context"
// 	"encoding/json"
// 	"strings"

// 	"github.com/drone/drone-yaml/yaml"
// 	"github.com/drone/drone/core"
// 	"github.com/drone/drone/plugin/registry/auths"
// )

// type registryManager struct {
// 	build   *core.Build
// 	config  *yaml.Manifest
// 	repo    *core.Repository
// 	auths   core.RegistryService
// 	secrets core.SecretService
// }

// func (s *registryManager) list(_ context.Context) ([]*core.Registry, error) {
// 	// get the registry credentials from the external
// 	// registry credential provider. This could, for example,
// 	// source credentials from ~/.docker/config.json
// 	registries, err := s.auths.List(noContext, &core.RegistryRequest{
// 		Repo:  s.repo,
// 		Build: s.build,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	// // get the registry credentials from the external
// 	// // user-defined registry credential provider.
// 	// userdef, err := s.auths.ListEndpoint(noContext, &core.RegistryRequest{
// 	// 	Repo:  s.repo,
// 	// 	Build: s.build,
// 	// }, s.repo.Endpoints.Registry)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }
// 	// // append user-defined registry credentials to the list.
// 	// registries = append(registries, userdef...)

// 	// the user can also define registry credentials in the
// 	// yaml secret section.
// 	for _, resource := range s.config.Resources {
// 		res, ok := resource.(*yaml.Secret)
// 		if !ok {
// 			continue
// 		}
// 		for name, value := range res.Data {
// 			// skip secrets the are intended for use with authenticating
// 			// to the docker registry and pulling private images.
// 			if isDockerConfig(name) == false {
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
// 				err = json.Unmarshal([]byte(plaintext), secret)
// 				if err != nil {
// 					return nil, err
// 				}
// 				parsed, err := auths.ParseString(secret.Data)
// 				if err != nil {
// 					return nil, err
// 				}
// 				registries = append(registries, parsed...)

// 			} else {
// 				// the user has the option of aliasing the
// 				// secret name. If the user specifies an external
// 				// name it must be used for the external query.
// 				req := &core.SecretRequest{
// 					Name:  value,
// 					Repo:  s.repo,
// 					Build: s.build,
// 				}

// 				//
// 				// TODO: bradrydzewski this should fetch from
// 				// the user-defined secrets.
// 				//
// 				secret, err := s.secrets.Find(noContext, req)
// 				if err != nil {
// 					return nil, err
// 				}
// 				parsed, err := auths.ParseString(secret.Data)
// 				if err != nil {
// 					return nil, err
// 				}
// 				registries = append(registries, parsed...)
// 			}
// 		}
// 	}
// 	return registries, nil
// }
