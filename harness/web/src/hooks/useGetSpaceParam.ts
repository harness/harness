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

import { useParams } from 'react-router-dom'
import type { CODEProps } from 'RouteDefinitions'
import { useAppContext } from 'AppContext'

/**
 * Get space parameter.
 * Space is passed differently in standalone and embedded modes. In standalone
 * mode, space is available from routing url (aka being passed as `:space`
 * using react-router). In embedded mode, Harness UI's routing always works with
 * `:accountId/orgs/:orgIdentifier/projects/:projectIdentifier` so we can't get
 * `space` from URL. As a result, we have to pass `space` via AppContext.
 * @returns space parameter.
 */
export function useGetSpaceParam() {
  const { space: spaceFromParams = '' } = useParams<CODEProps>()
  const { space } = useAppContext()
  return space || spaceFromParams
}
