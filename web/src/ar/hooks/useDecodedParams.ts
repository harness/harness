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

import { get } from 'lodash-es'
import { useParams } from 'react-router-dom'

type ParamsType<T> = T & { [K in keyof T]: string }

const transformPathParams = <T>(params: ParamsType<T>): T => {
  const decodedParams: ParamsType<T> = Object.keys(params).reduce(
    (acc, curr) => ({
      ...acc,
      [curr]: decodeURIComponent(get(params, curr, ''))
    }),
    {} as ParamsType<T>
  )
  return decodedParams
}

// Utility function to decode each parameter value
// Decode parameters and return the decoded object
export const useDecodedParams = <T>(): T => {
  const params = useParams<ParamsType<T>>()
  return transformPathParams<T>(params)
}
