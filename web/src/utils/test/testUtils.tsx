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

/*
 * This file contains utilities for testing.
 */
import React from 'react'
import { UseGetProps, UseGetReturn, RestfulProvider } from 'restful-react'
import { queryByAttribute } from '@testing-library/react'
import { compile } from 'path-to-regexp'
import { createMemoryHistory } from 'history'
import { Router, Route, Switch, useLocation, useHistory } from 'react-router-dom'
import qs from 'qs'
import { enableMapSet } from 'immer'
import { StringsContext } from 'framework/strings'
import './testUtils.module.scss'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type UnknownType = any

export type UseGetMockData<TData, TError = undefined, TQueryParams = undefined, TPathParams = undefined> = Required<
  UseGetProps<TData, TError, TQueryParams, TPathParams>
>['mock']

export interface UseGetMockDataWithMutateAndRefetch<T> extends UseGetMockData<T> {
  mutate: () => Record<string, unknown>
  refetch: () => Record<string, unknown>
}

export interface UseMutateMockData<TData, TRequestBody = unknown> {
  loading?: boolean
  mutate?: (data?: TRequestBody) => Promise<TData>
}

export type UseGetReturnData<TData, TError = undefined, TQueryParams = undefined, TPathParams = undefined> = Omit<
  UseGetReturn<TData, TError, TQueryParams, TPathParams>,
  'absolutePath' | 'cancel' | 'response'
>

export const findDialogContainer = (): HTMLElement | null => document.querySelector('.bp3-dialog')
export const findPopoverContainer = (): HTMLElement | null => document.querySelector('.bp3-popover-content')

export interface TestWrapperProps {
  path?: string
  pathParams?: Record<string, string | number>
  queryParams?: Record<string, unknown>
  enableBrowserView?: boolean
  stringsData?: Record<string, string>
  getString?(key: string): string
}

export const CurrentLocation = (): JSX.Element => {
  const location = useLocation()
  return (
    <div>
      <h1>Not Found</h1>
      <div data-testid="location">{`${location.pathname}${
        location.search ? `?${location.search.replace(/^\?/g, '')}` : ''
      }`}</div>
    </div>
  )
}

export interface BrowserViewProps {
  enable?: boolean
  children: React.ReactNode
}

export function BrowserView(props: BrowserViewProps): React.ReactElement {
  const { enable, children } = props
  const location = useLocation()
  const history = useHistory()

  if (!enable) {
    return <>{children}</>
  }

  function handlePathChange(e: React.ChangeEvent<HTMLInputElement>) {
    history.replace(e.currentTarget.value)
  }

  const search = location.search ? `?${location.search.replace(/^\?/, '')}` : ''

  return (
    <div className="browser">
      <div className="browser-header">
        <input className="browser-path" value={location.pathname + search} onChange={handlePathChange} />
      </div>
      <div className="browser-content">{children}</div>
    </div>
  )
}

export const TestWrapper: React.FC<TestWrapperProps> = props => {
  enableMapSet()
  const { path = '/', pathParams = {}, queryParams = {}, stringsData = {}, getString = (key: string) => key } = props

  const search = qs.stringify(queryParams, { addQueryPrefix: true })
  const routePath = compile(path)(pathParams) + search

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const history = React.useMemo(() => createMemoryHistory({ initialEntries: [routePath] }), [])

  /** TODO: Try fixing this later. This is causing some tests to fail */
  // React.useEffect(() => {
  //   history.replace(compile(path)(pathParams) + qs.stringify(queryParams, { addQueryPrefix: true }))
  //   // eslint-disable-next-line react-hooks/exhaustive-deps
  // }, [path, pathParams, queryParams])

  return (
    <StringsContext.Provider value={{ data: stringsData as UnknownType, getString }}>
      <Router history={history}>
        <RestfulProvider base="/">
          <BrowserView enable={props.enableBrowserView}>
            <Switch>
              <Route exact path={path}>
                {props.children}
              </Route>
              <Route>
                <CurrentLocation />
              </Route>
            </Switch>
          </BrowserView>
        </RestfulProvider>
      </Router>
    </StringsContext.Provider>
  )
}

export const queryByNameAttribute = (name: string, container: HTMLElement): HTMLElement | null =>
  queryByAttribute('name', container, name)

/**
 * Test utility to mock any import. It's better than jest.mock() because you can use
 * mockImport() inside tests.
 *
 * Sample:
 *
 *  mockImport('services/cf', {
 *    useGetFeatureFlag: () => ({
 *      data: mockFeatureFlag,
 *      loading: undefined,
 *      error: undefined,
 *      refetch: jest.fn()
 *     })
 *  })
 *
 * Mock an `export default`:
 *
 *  mockImport('@cf/components/FlagActivation/FlagActivation', {
 *     // FlagActivation is exported as `export default`
 *     default: function FlagActivation() {
 *       return <div>FlagActivation</div>
 *    }
 *  })
 *
 * @param moduleName
 * @param implementation
 */
export function mockImport(moduleName: string, implementation: Record<string, UnknownType>) {
  // eslint-disable-next-line @typescript-eslint/no-require-imports, @typescript-eslint/no-var-requires, global-require
  const module = require(moduleName)

  Object.keys(implementation).forEach(key => {
    module[key] = implementation?.[key]
  })
}
