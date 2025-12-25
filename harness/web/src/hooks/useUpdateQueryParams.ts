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

import { useLocation, useHistory } from 'react-router-dom'
import qs from 'qs'
import type { IStringifyOptions } from 'qs'

import { useQueryParams } from './useQueryParams'

export interface UseUpdateQueryParamsReturn<T> {
  updateQueryParams(values: T, options?: IStringifyOptions, replaceHistory?: boolean): void
  replaceQueryParams(values: T, options?: IStringifyOptions, replaceHistory?: boolean): void
}

export function useUpdateQueryParams<T = Record<string, string>>(
  globalOptions?: IStringifyOptions
): UseUpdateQueryParamsReturn<T> {
  const { pathname } = useLocation()
  const { push, replace } = useHistory()
  const queryParams = useQueryParams<T>()

  return {
    updateQueryParams(values: T, options?: IStringifyOptions, replaceHistory = false): void {
      const path = `${pathname}?${qs.stringify({ ...queryParams, ...values }, { ...globalOptions, ...options })}`
      replaceHistory ? replace(path) : push(path)
    },
    replaceQueryParams(values: T, options?: IStringifyOptions, replaceHistory = false): void {
      const path = `${pathname}?${qs.stringify(values, { ...globalOptions, ...options })}`
      replaceHistory ? replace(path) : push(path)
    }
  }
}
