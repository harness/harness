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

package githook

import "github.com/google/wire"

// Due to cyclic injection dependencies, wiring can be found at app/githook/wire.go

var WireSet = wire.NewSet(
	ProvidePreReceiveExtender,
	ProvideUpdateExtender,
	ProvidePostReceiveExtender,
)

func ProvidePreReceiveExtender() (PreReceiveExtender, error) {
	return NewPreReceiveExtender(), nil
}

func ProvideUpdateExtender() (UpdateExtender, error) {
	return NewUpdateExtender(), nil
}

func ProvidePostReceiveExtender() (PostReceiveExtender, error) {
	return NewPostReceiveExtender(), nil
}
