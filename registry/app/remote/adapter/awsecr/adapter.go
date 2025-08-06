// Source: https://github.com/goharbor/harbor

// Copyright 2016 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package awsecr

import (
	"context"
	"regexp"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	adp "github.com/harness/gitness/registry/app/remote/adapter"
	"github.com/harness/gitness/registry/app/remote/adapter/native"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	awsecrapi "github.com/aws/aws-sdk-go/service/ecr"
	"github.com/rs/zerolog/log"
)

const (
	//nolint:lll
	ecrPattern = "https://(?:api|(\\d+)\\.dkr)\\.ecr(\\-fips)?\\.([\\w\\-]+)\\.(amazonaws\\.com(\\.cn)?|sc2s\\.sgov\\.gov|c2s\\.ic\\.gov)"
)

var (
	ecrRegexp = regexp.MustCompile(ecrPattern)
)

func init() {
	adapterType := string(artifact.UpstreamConfigSourceAwsEcr)
	if err := adp.RegisterFactory(adapterType, new(factory)); err != nil {
		log.Error().Stack().Err(err).Msgf("Register adapter factory for %s", adapterType)
		return
	}
}

func newAdapter(
	ctx context.Context, spaceFinder refcache.SpaceFinder, service secret.Service, registry types.UpstreamProxy,
) (adp.Adapter, error) {
	accessKey, secretKey, isPublic, err := getCreds(ctx, spaceFinder, service, registry)
	if err != nil {
		return nil, err
	}
	var svc *awsecrapi.ECR
	if !isPublic {
		svc, err = getAwsSvc(ctx, accessKey, secretKey, registry)
		if err != nil {
			return nil, err
		}
	}

	authorizer := NewAuth(accessKey, svc, isPublic)

	return &adapter{
		cacheSvc: svc,
		Adapter:  native.NewAdapterWithAuthorizer(registry, authorizer),
	}, nil
}

// Create ...
func (f *factory) Create(
	ctx context.Context, spaceFinder refcache.SpaceFinder, record types.UpstreamProxy, service secret.Service,
) (adp.Adapter, error) {
	return newAdapter(ctx, spaceFinder, service, record)
}

type factory struct {
}

// HealthCheck checks health status of a proxy.
func (a *adapter) HealthCheck() (string, error) {
	return "Not implemented", nil
}

func (a *adapter) GetImageName(imageName string) (string, error) {
	return imageName, nil
}

var (
	_ adp.Adapter          = (*adapter)(nil)
	_ adp.ArtifactRegistry = (*adapter)(nil)
)

type adapter struct {
	*native.Adapter
	cacheSvc *awsecrapi.ECR
}

// Ensure '*adapter' implements interface 'Adapter'.
var _ adp.Adapter = (*adapter)(nil)
