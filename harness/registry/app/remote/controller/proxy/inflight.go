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

package proxy

import "sync"

type inflightRequest struct {
	mu     sync.Mutex
	reqMap map[string]interface{}
}

var inflightChecker = &inflightRequest{
	reqMap: make(map[string]interface{}),
}

// addRequest if the artifact already exist in the inflightRequest, return false
// else return true.
func (in *inflightRequest) addRequest(artifact string) (suc bool) {
	in.mu.Lock()
	defer in.mu.Unlock()
	_, ok := in.reqMap[artifact]
	if ok {
		// Skip some following operation if it is in reqMap.
		return false
	}
	in.reqMap[artifact] = 1
	return true
}

func (in *inflightRequest) removeRequest(artifact string) {
	in.mu.Lock()
	defer in.mu.Unlock()
	delete(in.reqMap, artifact)
}
