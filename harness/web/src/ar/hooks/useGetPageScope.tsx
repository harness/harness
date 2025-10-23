/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import type { Scope } from '@ar/MFEAppTypes'
import { EntityScope } from '@ar/common/types'
import { useAppStore } from './useAppStore'

export const getEntityScopeType = (scope: Scope): EntityScope => {
  return scope.projectIdentifier ? EntityScope.PROJECT : scope.orgIdentifier ? EntityScope.ORG : EntityScope.ACCOUNT
}

function useGetPageScope() {
  const { scope } = useAppStore()
  return getEntityScopeType(scope)
}

export default useGetPageScope
