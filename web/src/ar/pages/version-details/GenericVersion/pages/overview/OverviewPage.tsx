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
import { FontVariation } from '@harnessio/design-system'
import { Card, Container, Layout, Page, Text } from '@harnessio/uicore'
import { useGetArtifactDetailsQuery } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import { useDecodedParams, useGetSpaceRef } from '@ar/hooks'
import { VersionOverviewCard } from '@ar/pages/version-details/components/OverviewCards/types'
import VersionOverviewCards from '@ar/pages/version-details/components/OverviewCards/OverviewCards'
import { LabelValueContent } from '@ar/pages/version-details/components/LabelValueContent/LabelValueContent'

import type { GenericArtifactDetails } from '../../types'

import css from './styles.module.scss'
import genericStyles from '../../styles.module.scss'

export default function GenericOverviewPage() {
  const { getString } = useStrings()
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const spaceRef = useGetSpaceRef()

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetArtifactDetailsQuery({
    registry_ref: spaceRef,
    artifact: pathParams.artifactIdentifier,
    version: pathParams.versionIdentifier,
    queryParams: {}
  })

  const response = data?.content?.data as GenericArtifactDetails

  return (
    <Page.Body
      className={genericStyles.pageBody}
      loading={loading}
      error={error?.message || error}
      retryOnError={() => refetch()}>
      {response && (
        <Layout.Vertical className={genericStyles.cardContainer} spacing="medium">
          <VersionOverviewCards cards={[VersionOverviewCard.DEPLOYMENT, VersionOverviewCard.BUILD]} />
          <Card title={getString('versionDetails.overview.generalInformation.title')}>
            <Layout.Vertical spacing="medium">
              <Text font={{ variation: FontVariation.CARD_TITLE }}>
                {getString('versionDetails.overview.generalInformation.title')}
              </Text>
              <Container className={css.gridContainer}>
                <LabelValueContent
                  label={getString('versionDetails.overview.generalInformation.name')}
                  value={response.name}
                  withCopyText
                />
                <LabelValueContent
                  label={getString('versionDetails.overview.generalInformation.version')}
                  value={response.version}
                  withCopyText
                />
                <Text font={{ variation: FontVariation.SMALL_BOLD }}>
                  {getString('versionDetails.overview.generalInformation.packageType')}
                </Text>
                <Text icon="generic-repository-type" iconProps={{ size: 16 }} font={{ variation: FontVariation.SMALL }}>
                  {getString('packageTypes.genericPackage')}
                </Text>
                <LabelValueContent
                  label={getString('versionDetails.overview.generalInformation.uploadedBy')}
                  value={getReadableDateTime(Number(response.modifiedAt), DEFAULT_DATE_TIME_FORMAT)}
                />
                <LabelValueContent
                  label={getString('versionDetails.overview.generalInformation.description')}
                  value={defaultTo(response.description, getString('na'))}
                />
              </Container>
            </Layout.Vertical>
          </Card>
        </Layout.Vertical>
      )}
    </Page.Body>
  )
}
