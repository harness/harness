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

import React, { PropsWithChildren, Suspense, useEffect, useRef } from 'react'
import { Page } from '@harnessio/uicore'
import { HARServiceAPIClient } from '@harnessio/react-har-service-client'
import { QueryClientProvider } from '@tanstack/react-query'

import { StringsContextProvider } from '@ar/frameworks/strings/StringsContextProvider'
import { AppStoreContext } from '@ar/contexts/AppStoreContext'
import ParentProvider from '@ar/contexts/ParentProvider'
import type { ParentProviderProps } from '@ar/contexts/ParentProvider'
import { queryClient } from '@ar/utils/queryClient'

import { Parent } from '@ar/common/types'
import strings from '@ar/strings/strings.en.yaml'
import type { MFEAppProps } from '@ar/MFEAppTypes'
import DefaultNavComponent from '@ar/__mocks__/components/DefaultNavComponent'
import AppErrorBoundary from '@ar/components/AppErrorBoundary/AppErrorBoundary'

import css from '@ar/app/app.module.scss'

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
    on401
  } = props

  const { ModalProvider } = customComponents

  const appStoreData = React.useContext(parentContextObj.appStoreContext)
  useRef<HARServiceAPIClient>(
    new HARServiceAPIClient({
      responseInterceptor: (response: Response): Response => {
        if (!response.ok && response.status === 401) {
          on401()
        }

        return response
      },
      urlInterceptor: (url: string) => {
        return customUtils.getApiBaseUrl(url)
      },
      requestInterceptor(request) {
        request.headers.delete('Authorization')
        // add custom headers if available
        const customHeader = customUtils.getCustomHeaders()
        Object.entries(customHeader).map(([key, value]) => {
          request.headers.set(key, value)
        })
        return request
      }
    })
  )

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
            parent
          }}>
          <StringsContextProvider initialStrings={strings}>
            <ParentProvider
              hooks={{ ...hooks, ...customHooks } as ParentProviderProps['hooks']}
              components={{ ...components, ...customComponents } as ParentProviderProps['components']}
              utils={{ ...customUtils }}>
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
