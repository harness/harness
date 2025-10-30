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

import React, { PropsWithChildren, Suspense, useEffect } from 'react'
import { Page } from '@harnessio/uicore'
import { QueryClientProvider } from '@tanstack/react-query'

import { StringsContextProvider } from '@ar/frameworks/strings/StringsContextProvider'
import { AppStoreContext, RepositoryListViewTypeEnum } from '@ar/contexts/AppStoreContext'
import ParentProvider from '@ar/contexts/ParentProvider'
import type { ParentProviderProps } from '@ar/contexts/ParentProvider'
import { queryClient } from '@ar/utils/queryClient'

import { Parent } from '@ar/common/types'
import strings from '@ar/strings/strings.en.yaml'
import { PreferenceScope } from '@ar/constants'
import type { MFEAppProps } from '@ar/MFEAppTypes'
import PageNotPublic from '@ar/__mocks__/components/PageNotPublic'
import DefaultNavComponent from '@ar/__mocks__/components/DefaultNavComponent'
import AppErrorBoundary from '@ar/components/AppErrorBoundary/AppErrorBoundary'
import { useGovernanceMetaDataModal } from '@ar/__mocks__/hooks/useGovernanceMetaDataModal'

import useOpenApiClient from './useOpenApiClient'
import '@ar/utils/customYupValidators'

// Start: Add all factory registractions here
import '@ar/pages/version-details/VersionFactory'
import '@ar/pages/repository-details/RepositoryFactory'

import css from '@ar/app/app.module.scss'
import './themes.scss'

const RouteDestinations = React.lazy(() => import('@ar/routes/RouteDestinations'))

export default function ChildApp(props: PropsWithChildren<MFEAppProps>): React.ReactElement {
  const {
    renderUrl,
    parentContextObj,
    components,
    scope,
    customScope,
    hooks,
    customHooks,
    NavComponent = DefaultNavComponent,
    customComponents,
    parent,
    customUtils,
    matchPath,
    on401,
    isPublicAccessEnabledOnResources,
    isCurrentSessionPublic
  } = props

  const { ModalProvider } = customComponents
  const appStoreData = React.useContext(parentContextObj.appStoreContext)
  const { usePreferenceStore } = customHooks
  const { preference: repositoryListViewType, setPreference: setRepositoryListViewType } = usePreferenceStore<
    RepositoryListViewTypeEnum | undefined
  >(PreferenceScope.USER, 'RepositoryListViewType')

  useOpenApiClient({ on401, customUtils })

  useEffect(
    () => () => {
      if (typeof appStoreData.updateAppStore === 'function' && parent !== Parent.Enterprise) {
        appStoreData.updateAppStore({})
      }
    },
    []
  )

  return (
    <AppErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <AppStoreContext.Provider
          value={{
            ...appStoreData,
            matchPath,
            baseUrl: renderUrl,
            scope: { ...scope, ...customScope },
            repositoryListViewType: repositoryListViewType || RepositoryListViewTypeEnum.LIST,
            setRepositoryListViewType,
            parent,
            isPublicAccessEnabledOnResources,
            isCurrentSessionPublic
          }}>
          <StringsContextProvider initialStrings={strings}>
            <ParentProvider
              hooks={
                {
                  ...hooks,
                  ...customHooks,
                  useGovernanceMetaDataModal: customHooks.useGovernanceMetaDataModal ?? useGovernanceMetaDataModal // backward compatibility
                } as ParentProviderProps['hooks']
              }
              components={
                {
                  ...components,
                  ...customComponents,
                  PageNotPublic: customComponents.PageNotPublic ?? PageNotPublic // backward compatibility
                } as ParentProviderProps['components']
              }
              utils={{ ...customUtils }}
              contextObj={{ ...parentContextObj }}>
              <ModalProvider>
                {props.children ?? (
                  <NavComponent>
                    <Suspense
                      fallback={
                        <Page.Body className={css.pageBody}>
                          <Page.Spinner fixed={false} />
                        </Page.Body>
                      }>
                      <RouteDestinations />
                    </Suspense>
                  </NavComponent>
                )}
              </ModalProvider>
            </ParentProvider>
          </StringsContextProvider>
        </AppStoreContext.Provider>
      </QueryClientProvider>
    </AppErrorBoundary>
  )
}
