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

import { createContext } from 'react'
import type { AppstoreContext, Scope } from '@ar/MFEAppTypes'
import { Parent } from '@ar/common/types'

export interface ModuleAppStoreContextProps extends AppstoreContext, Record<string, unknown> {
  baseUrl: string
  matchPath: string
  scope: Scope & Record<string, string>
  parent: Parent
}

export const AppStoreContext = createContext<ModuleAppStoreContextProps>({
  currentUserInfo: { uuid: '' },
  featureFlags: {},
  updateAppStore: () => void 0,
  baseUrl: '',
  matchPath: '',
  scope: {},
  accountInfo: {},
  parent: Parent.OSS
})
