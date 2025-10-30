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

import React, { PropsWithChildren, createContext } from 'react'
import { noop } from 'lodash-es'

import type { AppstoreContext } from '@ar/MFEAppTypes'
import { useAppContext } from 'AppContext'

export const ArAppContext = createContext<AppstoreContext>({
  updateAppStore: noop,
  featureFlags: {}
})

interface ArAppProviderProps {
  initialValue?: Partial<AppstoreContext>
}

export function ArAppProvider({ children }: PropsWithChildren<ArAppProviderProps>) {
  const appContext = useAppContext()
  const { arAppStore } = appContext
  return (
    <ArAppContext.Provider
      value={{
        updateAppStore: val => {
          appContext.setAppContext({
            arAppStore: { ...arAppStore, ...val }
          })
        },
        featureFlags: FF_LIST
      }}>
      {children}
    </ArAppContext.Provider>
  )
}
