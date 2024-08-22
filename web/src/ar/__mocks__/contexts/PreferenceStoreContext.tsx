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

import React, { useEffect } from 'react'
import SecureStorage from '@ar/utils/SecureStorage'
import { useLocalStorage } from '@ar/hooks'

export enum PreferenceScope {
  USER = 'USER',
  MACHINE = 'MACHINE' // or workstation. This will act as default PreferenceScope
}

/**
 * Preference Store - helps to save ANY user-personalisation info
 */
export interface PreferenceStoreProps<T> {
  set(scope: PreferenceScope, entityToPersist: string, value: T): void
  get(scope: PreferenceScope, entityToRetrieve: string): T
  clear(scope: PreferenceScope, entityToRetrieve: string): void
}

export interface PreferenceStoreContextProps<T> {
  preference: T
  setPreference: (value: T) => void
  clearPreference: () => void
}

export const PREFERENCES_TOP_LEVEL_KEY = 'preferences'

export const PreferenceStoreContext = React.createContext<PreferenceStoreProps<any>>({
  set: /* istanbul ignore next */ () => void 0,
  get: /* istanbul ignore next */ () => void 0,
  clear: /* istanbul ignore next */ () => void 0
})

export function usePreferenceStore<T>(scope: PreferenceScope, entity: string): PreferenceStoreContextProps<T> {
  const { get, set, clear } = React.useContext(PreferenceStoreContext)

  const preference = get(scope, entity)
  const setPreference = set.bind(null, scope, entity)
  const clearPreference = clear.bind(null, scope, entity)

  return { preference, setPreference, clearPreference }
}

const checkAccess = (scope: PreferenceScope, contextArr: (string | undefined)[]): void => {
  if (!contextArr || contextArr?.some(val => val === undefined)) {
    const error = new Error(`PreferenceStore: Access to "${scope}" scope is not available in the current context.`)
    if (__DEV__) {
      console.error(error) // eslint-disable-line no-console
    }
  }
}

const getKey = (arr: (string | undefined)[], entity: string): string => {
  return [...arr, entity].join('/')
}

export const PreferenceStoreProvider: React.FC = (props: React.PropsWithChildren<unknown>) => {
  const [currentPreferences, setPreferences] = useLocalStorage<Record<string, unknown>>(PREFERENCES_TOP_LEVEL_KEY, {})
  const userEmail = SecureStorage.get('email') as string
  const [scopeToKeyMap, setScopeToKeyMap] = React.useState({
    [PreferenceScope.USER]: [userEmail],
    [PreferenceScope.MACHINE]: []
  })

  useEffect(() => {
    setScopeToKeyMap({
      [PreferenceScope.USER]: [userEmail],
      [PreferenceScope.MACHINE]: []
    })
  }, [userEmail])

  const setPreference = (key: string, value: unknown): void => {
    setPreferences(prevState => {
      return { ...prevState, [key]: value }
    })
  }

  const getPreference = (key: string): any => {
    return currentPreferences[key]
  }

  const clearPreference = (key: string): void => {
    const newPreferences = { ...currentPreferences }
    delete newPreferences[key]
    setPreferences(newPreferences)
  }

  const set = (scope: PreferenceScope, entityToPersist: string, value: unknown): void => {
    checkAccess(scope, scopeToKeyMap[scope])
    const key = getKey(scopeToKeyMap[scope], entityToPersist)
    setPreference(key, value)
  }

  const get = (scope: PreferenceScope, entityToRetrieve: string): unknown => {
    const key = getKey(scopeToKeyMap[scope], entityToRetrieve)
    return getPreference(key)
  }

  const clear = (scope: PreferenceScope, entityToRetrieve: string): void => {
    const key = getKey(scopeToKeyMap[scope], entityToRetrieve)
    clearPreference(key)
  }

  return (
    <PreferenceStoreContext.Provider
      value={{
        set,
        get,
        clear
      }}>
      {props.children}
    </PreferenceStoreContext.Provider>
  )
}
