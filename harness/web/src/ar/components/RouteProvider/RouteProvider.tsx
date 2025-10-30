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

import React from 'react'
import { Route } from 'react-router-dom'
import type { PropsWithChildren } from 'react'
import type { RouteProps } from 'react-router-dom'

import { useAppStore, useParentComponents } from '@ar/hooks'
import ParentSyncProvider from './ParentSyncProvider'

interface RouteProviderProps extends RouteProps {
  enabled?: boolean
  isPublic?: boolean
  onLoad?: (pathParams: Record<string, string>) => void
}

function RouteProvider(props: PropsWithChildren<RouteProviderProps>) {
  const { children, enabled = true, onLoad, isPublic, ...rest } = props
  const { ModalProvider, PageNotPublic } = useParentComponents()
  const { isCurrentSessionPublic } = useAppStore()
  if (isCurrentSessionPublic && !isPublic) {
    return <PageNotPublic />
  }
  if (!enabled) return <></>
  return (
    <Route {...rest}>
      <ParentSyncProvider onLoad={onLoad}>
        <ModalProvider>{children}</ModalProvider>
      </ParentSyncProvider>
    </Route>
  )
}

export default RouteProvider
