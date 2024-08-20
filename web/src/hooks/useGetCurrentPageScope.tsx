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
import { LabelsPageScope } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import type { Identifier } from 'utils/types'

export function useGetCurrentPageScope() {
  const { routingId: accountIdentifier, standalone } = useAppContext()
  const { orgIdentifier, projectIdentifier } = useParams<Identifier>()
  if (standalone) return LabelsPageScope.SPACE
  else if (projectIdentifier) return LabelsPageScope.PROJECT
  else {
    if (orgIdentifier) return LabelsPageScope.ORG
    else if (accountIdentifier) LabelsPageScope.ACCOUNT
  }
  return LabelsPageScope.ACCOUNT
}
