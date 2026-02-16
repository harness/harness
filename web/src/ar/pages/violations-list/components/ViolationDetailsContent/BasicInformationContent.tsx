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
import { Container, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import type { ArtifactScanDetails } from '@harnessio/react-har-service-client'

import { useRoutes } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryPackageType } from '@ar/common/types'
import { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'

import InformationMetrics from './InformationMetrics'
import css from './ViolationDetailsContent.module.scss'

interface BasicInformationContentProps {
  data: ArtifactScanDetails
}

function BasicInformationContent({ data }: BasicInformationContentProps) {
  const { getString } = useStrings()
  const routes = useRoutes()
  return (
    <Layout.Vertical spacing="large">
      <Text font={{ variation: FontVariation.H5, weight: 'bold' }} color={Color.GREY_700}>
        {getString('violationsList.violationDetailsModal.basicInformationSection.title')}
      </Text>
      <Container className={css.gridContainer}>
        <InformationMetrics.Link
          label={getString('violationsList.violationDetailsModal.basicInformationSection.packageName')}
          value={data.packageName}
          linkTo={routes.toARRedirect({
            packageType: data.packageType as RepositoryPackageType,
            registryId: data.registryName,
            artifactId: data.packageName,
            versionId: data.version,
            versionDetailsTab: VersionDetailsTab.OVERVIEW
          })}
        />
        <InformationMetrics.Link
          label={getString('violationsList.violationDetailsModal.basicInformationSection.upstreamProxy')}
          value={data.registryName}
          linkTo={routes.toARRepositoryDetails({
            repositoryIdentifier: data.registryName
          })}
        />
        <InformationMetrics.ScanStatus
          label={getString('violationsList.violationDetailsModal.basicInformationSection.status')}
          status={data.scanStatus}
          scanId={data.id}
        />
      </Container>
    </Layout.Vertical>
  )
}

export default BasicInformationContent
