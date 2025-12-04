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

package importer

import (
	"context"

	"github.com/harness/gitness/errors"
)

type ConnectorService interface {
	AsProvider(ctx context.Context, connector ConnectorDef) (Provider, error)
}

type connectorServiceNoop struct{}

func (connectorServiceNoop) AsProvider(context.Context, ConnectorDef) (Provider, error) {
	return Provider{}, errors.InvalidArgument("This feature is not supported.")
}
