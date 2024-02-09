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

package check

import (
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func ProvideCheckSanitizers() map[enum.CheckPayloadKind]func(in *ReportInput, s *auth.Session) error {
	registeredCheckSanitizers := make(map[enum.CheckPayloadKind]func(in *ReportInput, s *auth.Session) error)

	registeredCheckSanitizers[enum.CheckPayloadKindEmpty] = createEmptyPayloadSanitizer()

	registeredCheckSanitizers[enum.CheckPayloadKindRaw] = createRawPayloadSanitizer()

	// Markdown and Raw are the same.
	registeredCheckSanitizers[enum.CheckPayloadKindMarkdown] = registeredCheckSanitizers[enum.CheckPayloadKindRaw]

	registeredCheckSanitizers[enum.CheckPayloadKindPipeline] = createPipelinePayloadSanitizer()
	return registeredCheckSanitizers
}

func createEmptyPayloadSanitizer() func(in *ReportInput, _ *auth.Session) error {
	return func(in *ReportInput, _ *auth.Session) error {
		// the default payload kind (empty) does not support the payload data: clear it here
		in.Payload.Version = ""
		in.Payload.Data = []byte("{}")

		if in.Link == "" { // the link is mandatory as there is nothing in the payload
			return usererror.BadRequest("Link is missing")
		}

		return nil
	}
}

func createRawPayloadSanitizer() func(in *ReportInput, _ *auth.Session) error {
	return func(in *ReportInput, _ *auth.Session) error {
		// the text payload kinds (raw and markdown) do not support the version
		if in.Payload.Version != "" {
			return usererror.BadRequestf("Payload version must be empty for the payload kind '%s'",
				in.Payload.Kind)
		}

		payloadDataJSON, err := SanitizeJSONPayload(in.Payload.Data, &types.CheckPayloadText{})
		if err != nil {
			return err
		}

		in.Payload.Data = payloadDataJSON

		return nil
	}
}

func createPipelinePayloadSanitizer() func(in *ReportInput, _ *auth.Session) error {
	return func(_ *ReportInput, _ *auth.Session) error {
		return usererror.BadRequest("Kind cannot be pipeline for external checks")
	}
}
