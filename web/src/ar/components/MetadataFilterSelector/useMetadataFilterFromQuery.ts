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

import { useCallback } from 'react'
import type { MetadataType } from '@harnessio/react-har-service-client'

import { useParentHooks } from '@ar/hooks'

const MetadataFilterKeyPrefix = 'metadata.'
interface MetadataFilter {
  key: string
  value: string
  type: MetadataType
}

export default function useMetadatadataFilterFromQuery() {
  const { useQueryParams, useUpdateQueryParams } = useParentHooks()
  const queryParams = useQueryParams<Record<string, string>>()
  const { replaceQueryParams } = useUpdateQueryParams<Record<string, string>>()

  const getValue = useCallback((): MetadataFilter[] => {
    const value = Object.keys(queryParams)
      .filter(key => key.startsWith(MetadataFilterKeyPrefix))
      .reduce((acc, key) => {
        acc.push({
          key: key.replace(MetadataFilterKeyPrefix, ''),
          value: queryParams[key],
          type: 'MANUAL'
        })
        return acc
      }, [] as MetadataFilter[])
    return value
  }, [queryParams])

  const updateValue = useCallback(
    (val: MetadataFilter[]): void => {
      const newQueryParams = { ...queryParams }
      const metadataKeys = Object.keys(newQueryParams).filter(key => key.startsWith(MetadataFilterKeyPrefix))
      // remove old keys
      metadataKeys.forEach(key => {
        delete newQueryParams[key]
      })
      // add new keys
      val.forEach(v => {
        newQueryParams[`${MetadataFilterKeyPrefix}${v.key}`] = v.value
      })
      replaceQueryParams(newQueryParams)
    },
    [queryParams, replaceQueryParams]
  )

  const getValueForAPI = () => {
    const values = getValue()
    return values.map(v => `${v.key}:${v.value}`)
  }

  return {
    getValue,
    updateValue,
    getValueForAPI
  }
}
