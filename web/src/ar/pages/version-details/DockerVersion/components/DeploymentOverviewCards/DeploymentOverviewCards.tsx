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
import type { DeploymentStats, DockerManifestDetails } from '@harnessio/react-har-service-client'
import type { LayoutProps } from '@harnessio/uicore/dist/layouts/Layout'

import { useStrings } from '@ar/frameworks/strings'
import DeploymentsCard from '@ar/pages/version-details/components/DeploymentsCard/DeploymentsCard'

import css from './DeploymentOverviewCards.module.scss'

interface artifactDetails {
  artifactName: string
  version: string
  digests: DockerManifestDetails[]
}

interface DeploymentOverviewCardsProps extends LayoutProps {
  artifactDetails?: artifactDetails
  deploymentStats?: DeploymentStats
}

export default function DeploymentOverviewCards(props: DeploymentOverviewCardsProps) {
  const { artifactDetails, deploymentStats, ...rest } = props
  const { getString } = useStrings()
  return (
    <Layout.Horizontal width="100%" spacing="medium" {...rest}>
      <Card className={css.containerDetailsCard}>
        <Layout.Vertical spacing="large">
          <Text font={{ variation: FontVariation.CARD_TITLE }}>
            {getString('versionDetails.cards.container.title')}
          </Text>
          <Layout.Vertical spacing="small">
            <Text font={{ variation: FontVariation.BODY }}>
              {defaultTo(artifactDetails?.artifactName, getString('na'))}
            </Text>
            <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
              {getString('versionDetails.cards.container.versionDigest', {
                version: defaultTo(artifactDetails?.version, getString('na')),
                digest: defaultTo(artifactDetails?.digests?.length, 0)
              })}
            </Text>
          </Layout.Vertical>
        </Layout.Vertical>
      </Card>
      <DeploymentsCard
        prodCount={defaultTo(deploymentStats?.Production, 0)}
        nonProdCount={defaultTo(deploymentStats?.PreProduction, 0)}
      />
    </Layout.Horizontal>
  )
}
