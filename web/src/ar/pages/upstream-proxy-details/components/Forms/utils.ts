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

import produce from 'immer'
import { compact, defaultTo, get, set } from 'lodash-es'
import type { UserPassword } from '@harnessio/react-har-service-client'

import { Parent } from '@ar/common/types'
import type { Scope } from '@ar/MFEAppTypes'

import {
  DockerRepositoryURLInputSource,
  UpstreamProxyAuthenticationMode,
  type UpstreamRegistryRequest
} from '../../types'

export function getSecretSpacePath(referenceString: string, scope?: Scope) {
  if (!scope) return referenceString
  if (referenceString.startsWith('account.')) {
    return compact([scope.accountId]).join('/')
  }
  if (referenceString.startsWith('org.')) {
    return compact([scope.accountId, scope.orgIdentifier]).join('/')
  }
  return compact([scope.accountId, scope.orgIdentifier, scope.projectIdentifier]).join('/')
}

export function getReferenceStringFromSecretSpacePath(identifier: string, secretSpacePath: string) {
  const [accountId, orgIdentifier, projectIdentifier] = secretSpacePath.split('/')
  if (projectIdentifier) return identifier
  if (orgIdentifier) return `org.${identifier}`
  if (accountId) return `account.${identifier}`
  return identifier
}

export function getFormattedFormDataForAuthType(
  values: UpstreamRegistryRequest,
  parent?: Parent,
  scope?: Scope
): UpstreamRegistryRequest {
  return produce(values, (draft: UpstreamRegistryRequest) => {
    if (draft.config.authType === UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD) {
      set(draft, 'config.auth.authType', draft.config.authType)
      if (parent === Parent.Enterprise) {
        const auth = draft.config.auth as UserPassword
        const password = auth?.secretIdentifier
        set(draft, 'config.auth.secretSpacePath', getSecretSpacePath(get(password, 'referenceString', ''), scope))
        set(draft, 'config.auth.secretIdentifier', get(password, 'identifier'))
      }
    }
    if (draft.config.source !== DockerRepositoryURLInputSource.Custom) {
      set(draft, 'config.url', '')
    }
  })
}

function getSecretScopeDetailsByIdentifier(identifier: string, secretSpacePath: string) {
  const referenceString = getReferenceStringFromSecretSpacePath(identifier, secretSpacePath)
  const [, orgIdentifier, projectIdentifier] = secretSpacePath.split('/')
  return {
    identifier: identifier,
    name: identifier,
    referenceString,
    orgIdentifier,
    projectIdentifier
  }
}

export function getFormattedInitialValuesForAuthType(values: UpstreamRegistryRequest, parent?: Parent) {
  return produce(values, (draft: UpstreamRegistryRequest) => {
    if (draft.config.authType === UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD) {
      if (parent === Parent.Enterprise) {
        const auth = draft.config.auth as UserPassword
        const secretIdentifier = defaultTo(auth?.secretIdentifier, '')
        const secretSpacePath = defaultTo(auth?.secretSpacePath, '')
        set(draft, 'config.auth.secretIdentifier', getSecretScopeDetailsByIdentifier(secretIdentifier, secretSpacePath))
      }
    }
  })
}
