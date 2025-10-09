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
import { ScopeEnum } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import type { Identifier } from 'utils/types'

export function useGetCurrentPageScope(): ScopeEnum {
  const { routingId: accountIdentifier, standalone } = useAppContext()
  const { orgIdentifier, projectIdentifier, repoName } = useParams<Identifier & CODEProps>()

  if (repoName) return ScopeEnum.REPO_SCOPE
  if (standalone) return ScopeEnum.SPACE_SCOPE
  if (projectIdentifier) return ScopeEnum.PROJECT_SCOPE
  if (orgIdentifier) return ScopeEnum.ORG_SCOPE
  if (accountIdentifier) return ScopeEnum.ACCOUNT_SCOPE

  return ScopeEnum.ACCOUNT_SCOPE
}
