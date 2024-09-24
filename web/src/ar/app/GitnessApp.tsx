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

import React, { createContext } from 'react'
import { defaultTo, noop } from 'lodash-es'
import { Container } from '@harnessio/uicore'

import type { MFEAppProps } from '@ar/MFEAppTypes'

import {
  ModalProvider,
  useConfirmationDialog,
  useDefaultPaginationProps,
  useModalHook,
  useQueryParams,
  useQueryParamsOptions,
  useUpdateQueryParams
} from '@ar/__mocks__/hooks'
import RbacButton from '@ar/__mocks__/components/RbacButton'
import RbacMenuItem from '@ar/__mocks__/components/RbacMenuItem'
import NGBreadcrumbs from '@ar/__mocks__/components/NGBreadcrumbs'
import DependencyView from '@ar/__mocks__/components/DependencyView'
import SecretFormInput from '@ar/__mocks__/components/SecretFormInput'
import VulnerabilityView from '@ar/__mocks__/components/VulnerabilityView'
import { PreferenceStoreProvider, usePreferenceStore } from '@ar/__mocks__/contexts/PreferenceStoreContext'
import { Parent } from '@ar/common/types'
import App from '@ar/app/App'

import '@ar/styles/App.scss'
import getCustomHeaders from '@ar/__mocks__/utils/getCustomHeaders'
import { getApiBaseUrl } from '@ar/__mocks__/utils/getApiBaseUrl'

const GitnessApp = (props: Partial<MFEAppProps>): JSX.Element => {
  const {
    NavComponent,
    renderUrl,
    matchPath,
    scope,
    customScope,
    parentContextObj,
    components,
    hooks,
    customHooks,
    customComponents,
    customUtils,
    parent,
    on401
  } = props
  return (
    <Container className="arApp">
      <PreferenceStoreProvider>
        <App
          parent={defaultTo(parent, Parent.OSS)}
          parentContextObj={Object.assign(
            {
              appStoreContext: createContext({}) as any,
              licenseStoreProvider: createContext({}) as any,
              permissionsContext: createContext({}) as any
            },
            parentContextObj
          )}
          matchPath={defaultTo(matchPath, '/')}
          renderUrl={defaultTo(renderUrl, '/')}
          scope={defaultTo(scope, {})}
          customScope={defaultTo(customScope, {})}
          components={Object.assign(
            { RbacButton, NGBreadcrumbs, RbacMenuItem, SecretFormInput, VulnerabilityView, DependencyView },
            components
          )}
          NavComponent={NavComponent}
          hooks={Object.assign(
            {
              useDocumentTitle: () => ({ updateTitle: () => void 0 }),
              useLogout: () => ({ forceLogout: () => void 0 }),
              usePermission: () => [true]
            },
            hooks
          )}
          customHooks={Object.assign(
            {
              useQueryParams,
              useUpdateQueryParams,
              useQueryParamsOptions,
              useDefaultPaginationProps,
              usePreferenceStore,
              useModalHook,
              useConfirmationDialog
            },
            customHooks
          )}
          customComponents={Object.assign(
            {
              ModalProvider
            },
            customComponents
          )}
          customUtils={Object.assign(
            {
              getCustomHeaders,
              getApiBaseUrl
            },
            customUtils
          )}
          on401={defaultTo(on401, noop)}
        />
      </PreferenceStoreProvider>
    </Container>
  )
}

export default GitnessApp
