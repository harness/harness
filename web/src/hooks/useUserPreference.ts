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

import { useCallback, useState } from 'react'

export enum UserPreference {
  DIFF_VIEW_STYLE = 'DIFF_VIEW_STYLE',
  DIFF_LINE_BREAKS = 'DIFF_LINE_BREAKS',
  PULL_REQUEST_MERGE_STRATEGY = 'PULL_REQUEST_MERGE_STRATEGY',
  PULL_REQUEST_CREATION_OPTION = 'PULL_REQUEST_CREATION_OPTION',
  PULL_REQUEST_ACTIVITY_FILTER = 'PULL_REQUEST_ACTIVITY_FILTER',
  PULL_REQUEST_ACTIVITY_ORDER = 'PULL_REQUEST_ACTIVITY_ORDER'
}

export function useUserPreference<T = string>(
  key: UserPreference,
  defaultValue: T,
  filter: (val: T) => boolean = () => true
): [T, (val: T) => void, () => void] {
  const prefKey = `CODE_MOD_USER_PREF__${key}`
  const convert = useCallback(
    val => {
      if (val === undefined || val === null) {
        return val
      }

      if (typeof defaultValue === 'boolean') {
        return val === 'true'
      }

      if (typeof defaultValue === 'number') {
        return Number(val)
      }

      if (Array.isArray(defaultValue) || typeof defaultValue === 'object') {
        try {
          return JSON.parse(val)
        } catch (exception) {
          // eslint-disable-next-line no-console
          console.error('useUserPreference: Failed to parse object', val)
        }
      }
      return val
    },
    [defaultValue]
  )
  const [preference, setPreference] = useState<T>(convert(localStorage[prefKey]) || (defaultValue as T))
  const savePreference = useCallback(
    (val: T) => {
      try {
        if (filter(val)) {
          localStorage[prefKey] = Array.isArray(val) || typeof val === 'object' ? JSON.stringify(val) : val
        }
      } catch (exception) {
        // eslint-disable-next-line no-console
        console.error('useUserPreference: Failed to stringify object', val)
      }
      setPreference(val)
    },
    [prefKey, filter]
  )

  // NOTE: can be used to reset the value to the stored preference.
  const resetToPreference = useCallback(() => {
    setPreference(convert(localStorage[prefKey]) || (defaultValue as T))
  }, [prefKey, convert, defaultValue])

  return [preference, savePreference, resetToPreference]
}
