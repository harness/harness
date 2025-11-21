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
import { useGetDockerArtifactManifestQuery } from '@harnessio/react-har-service-client'

import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import { useDecodedParams, useGetSpaceRef, useParentHooks } from '@ar/hooks'

import ManifestDetails from '../components/ManifestDetails/ManifestDetails'
import type { DockerVersionDetailsQueryParams } from './types'

import css from './DockerVersion.module.scss'

export default function DockerManifestDetailsContent(): JSX.Element {
  const { useQueryParams } = useParentHooks()
  const { digest } = useQueryParams<DockerVersionDetailsQueryParams>()
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const spaceRef = useGetSpaceRef()

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetDockerArtifactManifestQuery(
    {
      registry_ref: spaceRef,
      artifact: encodeRef(pathParams.artifactIdentifier),
      version: pathParams.versionIdentifier,
      queryParams: {
        digest
      }
    },
    { enabled: !!digest }
  )

  const response = data?.content?.data
  return (
    <Page.Body
      className={css.pageBody}
      loading={loading || !digest}
      error={error?.message}
      retryOnError={() => refetch()}>
      <ManifestDetails className={css.manifestContainer} manifest={defaultTo(response?.manifest, '')} />
    </Page.Body>
  )
}
