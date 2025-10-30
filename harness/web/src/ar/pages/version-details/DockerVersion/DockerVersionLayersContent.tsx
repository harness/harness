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
import classNames from 'classnames'
import { FontVariation } from '@harnessio/design-system'
import { Card, Layout, Page, Text } from '@harnessio/uicore'
import { useGetDockerArtifactLayersQuery } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import { useDecodedParams, useGetSpaceRef, useParentHooks } from '@ar/hooks'
import LayersTable from './components/LayersTable/LayersTable'
import type { DockerVersionDetailsQueryParams } from './types'
import css from './DockerVersion.module.scss'

export default function DockerVersionLayersContent(): JSX.Element {
  const { getString } = useStrings()
  const { useQueryParams } = useParentHooks()
  const { digest } = useQueryParams<DockerVersionDetailsQueryParams>()
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const spaceRef = useGetSpaceRef()

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetDockerArtifactLayersQuery(
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
  const layers = data?.content.data.layers || []
  return (
    <Page.Body
      className={css.pageBody}
      loading={loading || !digest}
      error={error?.message}
      retryOnError={() => refetch()}>
      <Card className={classNames(css.cardContainer, css.margin0)}>
        <Layout.Vertical spacing="large">
          <Text font={{ variation: FontVariation.CARD_TITLE }}>
            {getString('versionDetails.artifactDetails.layers.imageLayers')}
          </Text>
          <LayersTable data={layers || []} />
        </Layout.Vertical>
      </Card>
    </Page.Body>
  )
}
