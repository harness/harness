// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package infraprovider

import (
	"context"
	"io"

	"github.com/harness/gitness/infraprovider/enum"
)

type InfraProvider interface {
	// Provision provisions infrastructure against a resourceKey with the provided parameters.
	Provision(ctx context.Context, spacePath string, resourceKey string, parameters []Parameter) (*Infrastructure, error)
	// Find finds infrastructure provisioned against a resourceKey.
	Find(ctx context.Context, spacePath string, resourceKey string, parameters []Parameter) (*Infrastructure, error)
	// Stop frees up the resources allocated against a resourceKey, which can be freed.
	Stop(ctx context.Context, infra *Infrastructure) (*Infrastructure, error)
	// Deprovision removes all infrastructure provisioned againest the resourceKey.
	Deprovision(ctx context.Context, infra *Infrastructure) (*Infrastructure, error)
	// AvailableParams provides a schema to define the infrastructure.
	AvailableParams() []ParameterSchema
	// ValidateParams validates the supplied params before defining the infrastructure resource .
	ValidateParams(parameters []Parameter) error
	// TemplateParams provides a list of params which are of type template.
	TemplateParams() []ParameterSchema
	// ProvisioningType specifies whether the provider will provision new infra resources or it will reuse existing.
	ProvisioningType() enum.InfraProvisioningType
	// Exec executes a shell command in the infrastructure.
	Exec(ctx context.Context, infra *Infrastructure, cmd []string) (io.Reader, io.Reader, error)
}
