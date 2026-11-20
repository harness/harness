// Copyright 2026 Harness, Inc.
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

package database

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
)

// logPrincipalLoadError logs a failed principal-info cache lookup. A cancelled request (client disconnect,
// aborted fetch) is expected rather than a server fault, so it is logged at debug level to keep it out of
// error-rate alerts and dashboards.
//
// The driver can report a cancelled query without wrapping context.Canceled, so the request context is
// checked as well.
func logPrincipalLoadError(ctx context.Context, err error, msg string) {
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		log.Ctx(ctx).Debug().Err(err).Msg(msg)
		return
	}

	log.Ctx(ctx).Err(err).Msg(msg)
}
