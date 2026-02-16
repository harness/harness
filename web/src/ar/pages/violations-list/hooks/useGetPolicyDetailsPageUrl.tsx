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

import { useAppStore } from '@ar/hooks/useAppStore'
import { useParentUtils } from '@ar/hooks/useParentUtils'
import type { Scope } from '@ar/MFEAppTypes'

function getScopeFromPolicyRef(policySetRef: string, scope: Scope): { scope: Scope; identifier: string } {
  if (policySetRef.startsWith('account.')) {
    return {
      scope: {
        accountId: scope.accountId
      },
      identifier: policySetRef.replace('account.', '')
    }
  } else if (policySetRef.startsWith('org.')) {
    return {
      scope: {
        accountId: scope.accountId,
        orgIdentifier: scope.orgIdentifier
      },
      identifier: policySetRef.replace('org.', '')
    }
  }
  return {
    scope: {
      accountId: scope.accountId,
      orgIdentifier: scope.orgIdentifier,
      projectIdentifier: scope.projectIdentifier
    },
    identifier: policySetRef
  }
}

export default function useGetPolicyDetailsPageUrl(policyRef: string) {
  const { scope } = useAppStore()
  const { routeToMode } = useParentUtils()
  const policyDetailsScope = getScopeFromPolicyRef(policyRef, scope)
  const basePath = routeToMode
    ? routeToMode({
        ...policyDetailsScope,
        module: 'har'
      })
    : ''
  return `${basePath}/settings/governance/policies/edit/${policyDetailsScope.identifier}`
}
