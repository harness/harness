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

import type { FC, PropsWithChildren } from 'react'
import React, { createContext } from 'react'
import type { MFEAppProps } from '@ar/MFEAppTypes'

export interface ParentProviderProps {
  hooks: MFEAppProps['customHooks'] & Required<MFEAppProps['hooks']>
  components: MFEAppProps['customComponents'] & Required<MFEAppProps['components']>
  utils: MFEAppProps['customUtils']
}

export const ParentProviderContext = createContext<ParentProviderProps>({} as ParentProviderProps)

const ParentProvider: FC<PropsWithChildren<ParentProviderProps>> = ({ children, hooks, components, utils }) => (
  <ParentProviderContext.Provider value={{ hooks, components, utils }}>{children}</ParentProviderContext.Provider>
)

export default ParentProvider
