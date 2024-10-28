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

import React from 'react'
import QueryString from 'qs'
import { noop } from 'lodash-es'
import { compile } from 'path-to-regexp'
import { Route, Router, Switch } from 'react-router-dom'
import { createMemoryHistory } from 'history'
import { PropsWithChildren, useMemo } from 'react'

import type { Scope } from '@ar/MFEAppTypes'
import { Parent } from '@ar/common/types'
import { ModalProvider } from '@ar/__mocks__/hooks'
import type { UseStringsReturn } from '@ar/frameworks/strings'
import { AppStoreContext } from '@ar/contexts/AppStoreContext'
import ParentProvider from '@ar/contexts/ParentProvider'
import type { ParentProviderProps } from '@ar/contexts/ParentProvider'
import { StringsContextProvider } from '@ar/frameworks/strings/StringsContextProvider'

import TestWrapper from 'utils/test/TestWrapper'
import { CurrentLocation } from 'utils/test/testUtils'
import {
  MockLicenseContext,
  MockParentAppStoreContext,
  MockPermissionsContext,
  MockTestUtils,
  MockTokenContext,
  MockTooltipContext
} from './utils'

interface TestWrapperProps {
  path?: string
  pathParams?: Record<string, string | number>
  queryParams?: Record<string, unknown>
  stringsData?: Record<string, string>
  getString?: UseStringsReturn['getString']
  featureFlags?: Record<string, boolean>
  baseUrl?: string
  matchPath?: string
  scope?: Scope & Record<string, string>
  parent?: Parent
}

export default function ArTestWrapper(props: PropsWithChildren<TestWrapperProps>) {
  const {
    queryParams,
    path = '',
    pathParams,
    stringsData = {},
    getString = (key: string) => key,
    featureFlags = {},
    baseUrl = '',
    matchPath = '',
    scope = {},
    parent = Parent.Enterprise
  } = props
  const search = QueryString.stringify(queryParams, { addQueryPrefix: true })
  const routePath = compile(path)(pathParams) + search
  const history = useMemo(() => createMemoryHistory({ initialEntries: [routePath] }), [])

  return (
    <TestWrapper path={path} queryParams={queryParams} pathParams={pathParams} isCurrentSessionPublic>
      <Router history={history}>
        <AppStoreContext.Provider
          value={{
            featureFlags,
            baseUrl,
            matchPath,
            scope,
            parent,
            updateAppStore: noop
          }}>
          <StringsContextProvider initialStrings={stringsData} getString={getString}>
            <ParentProvider
              hooks={MockTestUtils.hooks as ParentProviderProps['hooks']}
              components={MockTestUtils.components as ParentProviderProps['components']}
              utils={MockTestUtils.utils as ParentProviderProps['utils']}
              contextObj={{
                licenseStoreProvider: MockLicenseContext,
                appStoreContext: MockParentAppStoreContext,
                tooltipContext: MockTooltipContext,
                permissionsContext: MockPermissionsContext,
                tokenContext: MockTokenContext
              }}>
              <ModalProvider>
                <Switch>
                  <Route exact path={path}>
                    {props.children}
                  </Route>
                  <Route>
                    <CurrentLocation />
                  </Route>
                </Switch>
              </ModalProvider>
            </ParentProvider>
          </StringsContextProvider>
        </AppStoreContext.Provider>
      </Router>
    </TestWrapper>
  )
}
