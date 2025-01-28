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
import { Layout, Page } from '@harnessio/uicore'
import { type ArtifactDetail, useGetArtifactDetailsQuery } from '@harnessio/react-har-service-client'

import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useDecodedParams, useGetSpaceRef } from '@ar/hooks'
import type { VersionDetailsPathParams } from '@ar/routes/types'

import MavnGeneralInformationCard from './MavnGeneralInformationCard'

import genericStyles from '../../MavenVersion.module.scss'

export default function MavenArtifactOverviewPage() {
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const spaceRef = useGetSpaceRef()

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetArtifactDetailsQuery({
    registry_ref: spaceRef,
    artifact: encodeRef(pathParams.artifactIdentifier),
    version: pathParams.versionIdentifier,
    queryParams: {}
  })

  const response = data?.content?.data as ArtifactDetail

  return (
    <Page.Body
      className={genericStyles.pageBody}
      loading={loading}
      error={error?.message || error}
      retryOnError={() => refetch()}>
      {response && (
        <Layout.Vertical className={genericStyles.cardContainer} spacing="medium">
          <MavnGeneralInformationCard data={response} />
        </Layout.Vertical>
      )}
    </Page.Body>
  )
}
