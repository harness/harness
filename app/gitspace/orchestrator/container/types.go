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

package container

type StartResponse struct {
	ContainerID      string
	ContainerName    string
	PublishedPorts   map[int]string
	AbsoluteRepoPath string
	RemoteUser       string
}

type PostAction string

const (
	PostCreateAction PostAction = "post-create"
	PostStartAction  PostAction = "post-start"
)

type State string

const (
	ContainerStateRunning = State("running")
	ContainerStateRemoved = State("removed")
	ContainerStateDead    = State("dead")
	ContainerStateStopped = State("exited")
	ContainerStatePaused  = State("paused")
	ContainerStateUnknown = State("unknown")
	ContainerStateCreated = State("created")
)

type Action string

const (
	ContainerActionStop   = Action("stop")
	ContainerActionStart  = Action("start")
	ContainerActionRemove = Action("remove")
)
