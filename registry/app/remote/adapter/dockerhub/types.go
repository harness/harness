//  Copyright 2023 Harness, Inc.
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

package dockerhub

// LoginCredential is request to login.
type LoginCredential struct {
	User     string `json:"username"`
	Password string `json:"password"`
}

// TokenResp is response of login.
type TokenResp struct {
	Token string `json:"token"`
}

// NamespacesResp is namespace list responsed from DockerHub.
type NamespacesResp struct {
	// Namespaces is a list of namespaces
	Namespaces []string `json:"namespaces"`
}
