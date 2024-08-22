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
import { Layout } from '@harnessio/uicore'
import { useHistory } from 'react-router-dom'
import { useStrings } from '@ar/frameworks/strings'
import { useQueryParams } from '@ar/__mocks__/hooks'
import { useDecodedParams, useRoutes } from '@ar/hooks'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import DeploymentsCard from '@ar/pages/version-details/components/DeploymentsCard/DeploymentsCard'
import SecurityTestsCard from '@ar/pages/version-details/components/SecurityTestsCard/SecurityTestsCard'
import SupplyChainCard from '@ar/pages/version-details/components/SupplyChainCard/SupplyChainCard'
import { SecurityTestSatus } from '@ar/pages/version-details/components/SecurityTestsCard/types'
import { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'

import type { DockerVersionDetailsQueryParams } from '../../types'

import css from './OverviewCards.module.scss'

interface RedirectToTabOptions {
  sourceId?: string
  artifactId?: string
  executionIdentifier?: string
  pipelineIdentifier?: string
}

export default function DockerVersionOverviewCards() {
  const { getString } = useStrings()
  const routes = useRoutes()
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const { digest } = useQueryParams<DockerVersionDetailsQueryParams>()
  const history = useHistory()

  const handleRedirectToTab = (tab: VersionDetailsTab, options: RedirectToTabOptions = {}) => {
    let url = routes.toARVersionDetailsTab({
      ...pathParams,
      ...options,
      versionTab: tab
    })
    if (digest) {
      url = `${url}?digest=${digest}`
    }
    history.push(url)
  }

  return (
    <Layout.Horizontal width="100%" spacing="medium">
      <DeploymentsCard
        className={css.card}
        onClick={() => {
          handleRedirectToTab(VersionDetailsTab.DEPLOYMENTS)
        }}
        prodCount={10}
        nonProdCount={12}
        pipelineName="TestPipeline"
        executionId="1234566"
      />
      <SupplyChainCard
        onClick={() => {
          handleRedirectToTab(VersionDetailsTab.SUPPLY_CHAIN, {
            sourceId: '665f0d7575339c1dd7e4a885', // TODO: update this to correct values
            artifactId: '66901cc218bcea4482549235' // TODO: update this to correct values
          })
        }}
        className={css.card}
        totalComponents={10}
        allowListCount={1}
        denyListCount={2}
        sbomScore={9.5}
      />
      <SecurityTestsCard
        className={classNames(css.card, css.securityTestsCard)}
        onClick={() => {
          handleRedirectToTab(VersionDetailsTab.SECURITY_TESTS, {
            executionIdentifier: 'ZeRGcQCMSwu1cQhRQuFKFg', // TODO: Update this to correct values
            pipelineIdentifier: 'SSCA_STO_Clone' // TODO: Update this to correct values
          })
        }}
        totalCount={11}
        items={[
          {
            title: getString('versionDetails.cards.securityTests.critical'),
            value: 2,
            status: SecurityTestSatus.Critical
          },
          {
            title: getString('versionDetails.cards.securityTests.high'),
            value: 3,
            status: SecurityTestSatus.High
          },
          {
            title: getString('versionDetails.cards.securityTests.medium'),
            value: 5,
            status: SecurityTestSatus.Medium
          },
          {
            title: getString('versionDetails.cards.securityTests.low'),
            value: 1,
            status: SecurityTestSatus.Low
          }
        ]}
      />
    </Layout.Horizontal>
  )
}
