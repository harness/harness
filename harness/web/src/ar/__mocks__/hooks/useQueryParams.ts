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

import React, { useEffect, useMemo, useRef } from 'react'
import { useLocation } from 'react-router-dom'
import qs from 'qs'
import type { IParseOptions } from 'qs'
import { assignWith, get, isNil, set } from 'lodash-es'

export interface UseQueryParamsOptions<T> extends IParseOptions {
  processQueryParams?(data: any): T
}

export function useQueryParams<T = unknown>(options?: UseQueryParamsOptions<T>): T {
  const { search } = useLocation()

  const queryParams = React.useMemo(() => {
    const params = qs.parse(search, { ignoreQueryPrefix: true, ...options })

    if (typeof options?.processQueryParams === 'function') {
      return options.processQueryParams(params)
    }

    return params
  }, [search, options])

  return queryParams as unknown as T
}

type CustomQsDecoderOptions = {
  parseNumbers?: boolean
  parseBoolean?: boolean
  ignoreNull?: boolean
  ignoreEmptyString?: boolean
}

type CustomQsDecoder = (customQsDecoderOptions?: CustomQsDecoderOptions) => IParseOptions['decoder']

/**
 * By default, all values are parsed as strings by qs, except for arrays and objects
 * This is optional decoder that automatically transforms to numbers, booleans and null
 */
export const queryParamDecodeAll: CustomQsDecoder =
  ({ parseNumbers = true, parseBoolean = true, ignoreNull = true, ignoreEmptyString = true } = {}) =>
  (value, decoder) => {
    if (parseNumbers && /^(\d+|\d*\.\d+)$/.test(value)) {
      return parseFloat(value)
    }

    if (ignoreEmptyString && value.length === 0) {
      return
    }

    const keywords: Record<string, null | undefined> = {
      null: ignoreNull ? undefined : null,
      undefined: undefined
    }

    if (value in keywords) {
      return keywords[value]
    }

    const booleanKeywords: Record<string, boolean> = {
      true: true,
      false: false
    }

    if (parseBoolean && value in booleanKeywords) {
      return booleanKeywords[value]
    }

    return decoder(value)
  }

// list of params that should be converted back to strings
// if searchTerm is '123', queryParamDecodeAll converts it to a number, but searchTerm should remain a string
const ignoreList = ['searchTerm']

// This uses queryParamDecodeAll as the decoder and assigns the value from default params if the processed param's value is null/undefined.
export const useQueryParamsOptions = <Q extends object, DKey extends keyof Q>(
  defaultParams: { [K in DKey]: NonNullable<Q[K]> },
  decoderOptions?: CustomQsDecoderOptions
): UseQueryParamsOptions<Required<Pick<Q, DKey>>> => {
  const defaultParamsRef = useRef(defaultParams)
  useEffect(() => {
    defaultParamsRef.current = defaultParams
  }, [defaultParams])

  const options = useMemo(
    () => ({
      decoder: queryParamDecodeAll(decoderOptions),
      processQueryParams: (params: Q) => {
        const processedParams = { ...params }

        ignoreList.forEach(param => {
          if (!isNil(get(processedParams, param))) {
            set(processedParams, param, get(processedParams, param).toString())
          }
        })

        return assignWith(processedParams, defaultParamsRef.current, (objValue, srcValue) =>
          isNil(objValue) ? srcValue : objValue
        ) as Required<Pick<Q, DKey>>
      }
    }),
    []
  )
  return options
}
