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

package auth

import (
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// AnonymousPrincipal is an in-memory principal for users with no auth data.
// Authorizer is in charge of handling anonymous access.
var AnonymousPrincipal = types.Principal{
	ID:   -1,
	UID:  types.AnonymousPrincipalUID,
	Type: enum.PrincipalTypeUser,
}

func IsAnonymousSession(session *Session) bool {
	return session != nil && session.Principal.UID == types.AnonymousPrincipalUID
}
