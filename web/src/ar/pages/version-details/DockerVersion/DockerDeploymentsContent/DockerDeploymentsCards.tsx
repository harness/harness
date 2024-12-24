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
import { Card, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useGetDockerArtifactManifestsQuery } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useDecodedParams, useGetSpaceRef } from '@ar/hooks'
import type { VersionDetailsPathParams } from '@ar/routes/types'

import css from '../DockerVersion.module.scss'

export default function DockerDeploymentsCard() {
  const { getString } = useStrings()
  const registryRef = useGetSpaceRef()
  const params = useDecodedParams<VersionDetailsPathParams>()

  const { data: manifestsData } = useGetDockerArtifactManifestsQuery({
    registry_ref: registryRef,
    artifact: encodeRef(params.artifactIdentifier),
    version: params.versionIdentifier
  })
  return (
    <Card className={css.containerDetailsCard}>
      <Layout.Vertical spacing="large">
        <Text font={{ variation: FontVariation.CARD_TITLE }}>{getString('versionDetails.cards.container.title')}</Text>
        <Layout.Vertical spacing="small">
          <Text lineClamp={1} font={{ variation: FontVariation.BODY }}>
            {defaultTo(params.artifactIdentifier, getString('na'))}
          </Text>
          <Text lineClamp={1} color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
            {getString('versionDetails.cards.container.versionDigest', {
              version: defaultTo(params.versionIdentifier, getString('na')),
              digest: defaultTo(manifestsData?.content.data.manifests?.length, 0)
            })}
          </Text>
        </Layout.Vertical>
      </Layout.Vertical>
    </Card>
  )
}
