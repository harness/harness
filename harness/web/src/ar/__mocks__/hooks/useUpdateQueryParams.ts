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

import { useCallback, useEffect, useRef } from 'react'
import qs from 'qs'
import { isEmpty } from 'lodash-es'
import type { IStringifyOptions } from 'qs'
import { useLocation, useHistory } from 'react-router-dom'

import { useQueryParams } from './useQueryParams'

export interface UseUpdateQueryParamsReturn<T> {
  updateQueryParams(values: T, options?: IStringifyOptions, replaceHistory?: boolean): void
  replaceQueryParams(values: T, options?: IStringifyOptions, replaceHistory?: boolean): void
}

export function useUpdateQueryParams<T = Record<string, string>>(): UseUpdateQueryParamsReturn<T> {
  const { pathname } = useLocation()
  const { push, replace } = useHistory()
  const queryParams = useQueryParams<T>()

  // queryParams, pathname are stored in refs so that
  // updateQueryParams/replaceQueryParams can be memoized without changing too often
  const ref = useRef({ queryParams, pathname })
  useEffect(() => {
    ref.current = {
      queryParams,
      pathname
    }
  }, [queryParams, pathname])

  return {
    updateQueryParams: useCallback(
      (values: T, options?: IStringifyOptions, replaceHistory?: boolean): void => {
        const path = `${ref.current.pathname}?${qs.stringify({ ...ref.current.queryParams, ...values }, options)}`
        replaceHistory ? replace(path) : push(path)
      },
      [push, replace]
    ),
    replaceQueryParams: useCallback(
      (values: T, options?: IStringifyOptions, replaceHistory?: boolean): void => {
        if (isEmpty(values)) {
          ref.current = {
            ...ref.current,
            queryParams: {} as T
          }
        }
        const path = `${ref.current.pathname}?${qs.stringify(values, options)}`
        replaceHistory ? replace(path) : push(path)
      },
      [push, replace]
    )
  }
}
