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
import { defaultTo } from 'lodash-es'
import { useHistory } from 'react-router-dom'
import { Layout, Page } from '@harnessio/uicore'
import { useGetDockerArtifactIntegrationDetailsQuery } from '@harnessio/react-har-service-client'

import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useStrings } from '@ar/frameworks/strings'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import { useDecodedParams, useGetSpaceRef, useParentHooks, useRoutes } from '@ar/hooks'
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
  const { useQueryParams } = useParentHooks()
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const { digest } = useQueryParams<DockerVersionDetailsQueryParams>()
  const history = useHistory()
  const spaceRef = useGetSpaceRef()

  const { data, isFetching, error, refetch } = useGetDockerArtifactIntegrationDetailsQuery({
    registry_ref: spaceRef,
    artifact: encodeRef(pathParams.artifactIdentifier),
    version: pathParams.versionIdentifier,
    queryParams: {
      digest
    }
  })

  const responseData = data?.content.data

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
    <Page.Body
      className={css.container}
      loading={isFetching}
      retryOnError={() => refetch()}
      error={typeof error === 'string' ? error : error?.message}>
      {responseData && (
        <Layout.Horizontal width="100%" spacing="medium">
          <DeploymentsCard
            className={css.card}
            onClick={() => {
              handleRedirectToTab(VersionDetailsTab.DEPLOYMENTS)
            }}
            prodCount={defaultTo(responseData.deploymentsDetails?.prodDeployment, 0)}
            nonProdCount={defaultTo(responseData.deploymentsDetails?.nonProdDeployment, 0)}
            pipelineName={responseData.buildDetails?.pipelineDisplayName}
            pipelineId={responseData.buildDetails?.pipelineIdentifier}
            executionId={responseData.buildDetails?.pipelineExecutionId}
          />
          <SupplyChainCard
            onClick={() => {
              handleRedirectToTab(VersionDetailsTab.SUPPLY_CHAIN, {
                sourceId: responseData.sbomDetails?.artifactSourceId,
                artifactId: responseData.sbomDetails?.artifactId
              })
            }}
            className={css.card}
            totalComponents={defaultTo(responseData.sbomDetails?.componentsCount, 0)}
            allowListCount={defaultTo(responseData.sbomDetails?.allowListViolations, 0)}
            denyListCount={defaultTo(responseData.sbomDetails?.denyListViolations, 0)}
            sbomScore={defaultTo(responseData.sbomDetails?.avgScore, 0)}
          />
          <SecurityTestsCard
            className={classNames(css.card, css.securityTestsCard)}
            onClick={() => {
              handleRedirectToTab(VersionDetailsTab.SECURITY_TESTS, {
                executionIdentifier: responseData?.stoDetails?.executionId,
                pipelineIdentifier: responseData?.stoDetails?.pipelineId
              })
            }}
            totalCount={defaultTo(responseData.stoDetails?.total, 0)}
            items={[
              {
                title: getString('versionDetails.cards.securityTests.critical'),
                value: defaultTo(responseData.stoDetails?.critical, 0),
                status: SecurityTestSatus.Critical
              },
              {
                title: getString('versionDetails.cards.securityTests.high'),
                value: defaultTo(responseData.stoDetails?.high, 0),
                status: SecurityTestSatus.High
              },
              {
                title: getString('versionDetails.cards.securityTests.medium'),
                value: defaultTo(responseData.stoDetails?.medium, 0),
                status: SecurityTestSatus.Medium
              },
              {
                title: getString('versionDetails.cards.securityTests.low'),
                value: defaultTo(responseData.stoDetails?.low, 0),
                status: SecurityTestSatus.Low
              }
            ]}
          />
        </Layout.Horizontal>
      )}
    </Page.Body>
  )
}
