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

import qs, { IStringifyOptions } from 'qs'

export const getConfig = (str: string): string => {
  // 'code/api/v1' -> 'api/v1'       (standalone)
  //               -> 'code/api/v1'  (embedded inside Harness platform)
  if (window.STRIP_CODE_PREFIX) {
    str = str.replace(/^code\//, '')
  }

  return window.apiUrl ? `${window.apiUrl}/${str}` : `${window.harnessNameSpace || ''}/${str}`
}

export interface GetUsingFetchProps<
  _TData = any,
  _TError = any,
  TQueryParams = {
    [key: string]: any
  },
  TPathParams = {
    [key: string]: any
  }
> {
  queryParams?: TQueryParams
  queryParamStringifyOptions?: IStringifyOptions
  pathParams?: TPathParams
  requestOptions?: RequestInit
  mock?: _TData
}

export const getUsingFetch = <
  TData = any,
  _TError = any,
  TQueryParams = {
    [key: string]: any
  },
  TPathParams = {
    [key: string]: any
  }
>(
  base: string,
  path: string,
  props: GetUsingFetchProps<TData, _TError, TQueryParams, TPathParams>,
  signal?: RequestInit['signal']
): Promise<TData> => {
  if (props.mock) return Promise.resolve(props.mock)
  let url = base + path
  if (props.queryParams && Object.keys(props.queryParams).length) {
    url += `?${qs.stringify(props.queryParams, props.queryParamStringifyOptions)}`
  }
  return fetch(url, {
    signal,
    ...(props.requestOptions || {})
    // headers: getHeaders(props.requestOptions?.headers)
  }).then(res => {
    // custom event to allow the app framework to handle api responses
    const responseEvent = new CustomEvent('PROMISE_API_RESPONSE', { detail: { response: res } })
    window.dispatchEvent(responseEvent) // this will be captured in App.tsx to handle 401 and token refresh

    const contentType = res.headers.get('content-type') || ''

    if (contentType.toLowerCase().indexOf('application/json') > -1) {
      if (res.status === 401) {
        return res.json().then(json => Promise.reject(json))
      }
      return res.json()
    }

    if (res.status === 401) {
      return res.text().then(text => Promise.reject(text))
    }

    return res.text()
  })
}
