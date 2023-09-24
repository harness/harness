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

import { ExecutionState } from 'components/ExecutionStatus/ExecutionStatus'

export const getStatus = (status: string | undefined): ExecutionState => {
  switch (status) {
    case 'success':
      return ExecutionState.SUCCESS
    case 'failure':
      return ExecutionState.FAILURE
    case 'running':
      return ExecutionState.RUNNING
    case 'pending':
      return ExecutionState.PENDING
    case 'error':
      return ExecutionState.ERROR
    case 'killed':
      return ExecutionState.KILLED
    case 'skipped':
      return ExecutionState.SKIPPED
    default:
      return ExecutionState.PENDING
  }
}
