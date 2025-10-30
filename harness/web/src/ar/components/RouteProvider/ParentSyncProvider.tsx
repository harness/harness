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
import type { PropsWithChildren } from 'react'

import { Parent } from '@ar/common/types'
import { useAppStore, useDecodedParams, useDeepCompareEffect } from '@ar/hooks'

interface ParentSyncProviderProps {
  onLoad?: (pathParams: Record<string, string>) => void
}

export default function ParentSyncProvider(props: PropsWithChildren<ParentSyncProviderProps>) {
  const pathParams = useDecodedParams<Record<string, string>>()
  const { updateAppStore, parent } = useAppStore()

  useDeepCompareEffect(() => {
    if (typeof props.onLoad === 'function') {
      props.onLoad(pathParams)
    }
    if (typeof updateAppStore === 'function' && parent !== Parent.Enterprise) {
      updateAppStore({
        repositoryIdentifier: pathParams?.repositoryIdentifier as string,
        artifactIdentifier: pathParams?.artifactIdentifier as string,
        versionIdentifier: pathParams?.versionIdentifier as string
      })
    }
  }, [pathParams, parent])

  return <>{props.children}</>
}
