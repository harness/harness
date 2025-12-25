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
import { Layout } from '@harnessio/uicore'
import type { DeploymentStats } from '@harnessio/react-har-service-client'
import type { LayoutProps } from '@harnessio/uicore/dist/layouts/Layout'

import DeploymentsCard from '@ar/pages/version-details/components/DeploymentsCard/DeploymentsCard'

interface DeploymentOverviewCardsProps extends LayoutProps {
  prefixCards?: React.ReactNode
  deploymentStats?: DeploymentStats
}

export default function DeploymentOverviewCards(props: DeploymentOverviewCardsProps) {
  const { deploymentStats, prefixCards, ...rest } = props
  return (
    <Layout.Horizontal width="100%" spacing="medium" {...rest}>
      {prefixCards}
      <DeploymentsCard
        prodCount={defaultTo(deploymentStats?.Production, 0)}
        nonProdCount={defaultTo(deploymentStats?.PreProduction, 0)}
        hideBuildDetails
      />
    </Layout.Horizontal>
  )
}
