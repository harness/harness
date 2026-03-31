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

import React, { useMemo } from 'react'

import type { VersionDetailsTabPathParams } from '@ar/routes/types'
import { useDecodedParams, useParentHooks } from '@ar/hooks'
import { VersionDetails } from './VersionDetails'
import VersionProvider from './context/VersionProvider'

import './VersionFactory'

export default function VersionDetailsPage(): JSX.Element {
  const pathParams = useDecodedParams<VersionDetailsTabPathParams>()
  const { useDocumentTitle } = useParentHooks()

  const versionDetailsDocumentTitle = useMemo(
    () => [`${pathParams.artifactIdentifier}@(${pathParams.versionIdentifier})`, pathParams.versionTab || ''],
    [pathParams.artifactIdentifier, pathParams.versionIdentifier, pathParams.versionTab]
  )

  useDocumentTitle(versionDetailsDocumentTitle)

  return (
    <VersionProvider
      repoKey={pathParams.repositoryIdentifier}
      artifactKey={pathParams.artifactIdentifier}
      versionKey={pathParams.versionIdentifier}>
      <VersionDetails />
    </VersionProvider>
  )
}
