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
import { defaultTo } from 'lodash-es'
import { Page } from '@harnessio/uicore'
import { useGetHelmArtifactManifestQuery } from '@harnessio/react-har-service-client'

import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useDecodedParams, useGetSpaceRef } from '@ar/hooks'
import type { VersionDetailsPathParams } from '@ar/routes/types'

import ManifestDetails from '../components/ManifestDetails/ManifestDetails'

import css from './HelmVersion.module.scss'

export default function HelmManifestDetailsContent(): JSX.Element {
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const spaceRef = useGetSpaceRef()

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetHelmArtifactManifestQuery({
    registry_ref: spaceRef,
    artifact: encodeRef(pathParams.artifactIdentifier),
    version: pathParams.versionIdentifier
  })

  const response = data?.content?.data
  return (
    <Page.Body className={css.pageBody} loading={loading} error={error?.message} retryOnError={() => refetch()}>
      <ManifestDetails className={css.cardContainer} manifest={defaultTo(response?.manifest, '')} />
    </Page.Body>
  )
}
