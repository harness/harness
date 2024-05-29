/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

export enum GitspaceStatus {
  RUNNING = 'RUNNING',
  STOPPED = 'STOPPED',
  UNKNOWN = 'UNKNOWN',
  ERROR = 'ERROR'
}

export enum IDEType {
  VSCODE = 'VSCODE',
  VSCODEWEB = 'VSCODEWEB'
}

export enum GitspaceActionType {
  START = 'START',
  STOP = 'STOP'
}

export enum GitspaceRegion {
  USEast = 'US East',
  USWest = 'US West',
  Europe = 'Europe',
  Australia = 'Australia'
}

export enum CodeRepoAccessType {
  PRIVATE = 'PRIVATE',
  PUBLIC = 'PUBLIC'
}
