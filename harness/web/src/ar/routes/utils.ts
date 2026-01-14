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

import { defaultTo, omit } from 'lodash-es'
import QueryString from 'qs'

import { RepositoryListViewTypeEnum } from '@ar/contexts/AppStoreContext'

export function normalizePath(url: string): string {
  return url.replace(/\/{2,}/g, '/')
}

export const encodePathParams = (val?: string): string => {
  if (typeof val === 'string' && val.startsWith(':')) return val
  return encodeURIComponent(defaultTo(val, ''))
}

export interface IRouteOptions {
  queryParams?: Record<string, string>
  mode?: RepositoryListViewTypeEnum
  currentQueryParams?: Record<string, string>
}

export function routeDefinitionWithMode<T>(fn: (params: T) => string) {
  return (params: T, options?: IRouteOptions) => {
    const { queryParams, mode, currentQueryParams } = options || {}
    let finalQueryParams = queryParams
    let route = fn(params)
    // TODO: find a way to make this flexible
    if (mode === RepositoryListViewTypeEnum.DIRECTORY && currentQueryParams) {
      finalQueryParams = { ...finalQueryParams, ...omit(currentQueryParams, 'digest') }
    }
    if (finalQueryParams) {
      const queryString = QueryString.stringify(finalQueryParams, { encode: false })
      route = `${route}?${queryString}`
    }
    return route
  }
}
