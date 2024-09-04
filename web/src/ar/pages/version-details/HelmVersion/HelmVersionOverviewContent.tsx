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
import { useGetHelmArtifactDetailsQuery } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import { useDecodedParams, useGetSpaceRef } from '@ar/hooks'

import HelmVersionOverviewCards from './components/OverviewCards/OverviewCards'
import { LabelValueContent } from '../components/LabelValueContent/LabelValueContent'

import css from './HelmVersion.module.scss'

export default function HelmVersionOverviewContent(): JSX.Element {
  const { getString } = useStrings()
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const spaceRef = useGetSpaceRef()

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetHelmArtifactDetailsQuery({
    registry_ref: spaceRef,
    artifact: encodeRef(pathParams.artifactIdentifier),
    version: pathParams.versionIdentifier
  })

  const response = data?.content?.data

  return (
    <Page.Body className={css.pageBody} loading={loading} error={error?.message} retryOnError={() => refetch()}>
      {response && (
        <Layout.Vertical className={css.cardContainer} spacing="medium">
          <HelmVersionOverviewCards />
          <Card title="General Information">
            <Layout.Vertical spacing="medium">
              <Text font={{ variation: FontVariation.CARD_TITLE }}>
                {getString('versionDetails.overview.generalInformation.title')}
              </Text>
              <Container className={css.gridContainer}>
                <LabelValueContent
                  label={getString('versionDetails.overview.generalInformation.name')}
                  value={response.artifact}
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
                <Text icon="service-helm" iconProps={{ size: 16 }} font={{ variation: FontVariation.SMALL }}>
                  {getString('packageTypes.helmPackage')}
                </Text>
                <LabelValueContent
                  label={getString('versionDetails.overview.generalInformation.repositoryPath')}
                  value={response.registryPath}
                  withCopyText
                />
                <LabelValueContent
                  label={getString('versionDetails.overview.generalInformation.url')}
                  value={response.url}
                  withCopyText
                />
                <LabelValueContent
                  label={getString('versionDetails.overview.generalInformation.size')}
                  value={response.size}
                />
                <LabelValueContent
                  label={getString('versionDetails.overview.generalInformation.downloads')}
                  value={defaultTo(response.downloadsCount?.toLocaleString(), 0)}
                />
                <LabelValueContent
                  label={getString('versionDetails.overview.generalInformation.uploadedBy')}
                  value={getReadableDateTime(Number(response.modifiedAt), DEFAULT_DATE_TIME_FORMAT)}
                />
                <LabelValueContent
                  label={getString('versionDetails.overview.generalInformation.createdAndLastModifiedAt')}
                  value={getReadableDateTime(Number(response.createdAt), DEFAULT_DATE_TIME_FORMAT)}
                />
                <LabelValueContent
                  label={getString('versionDetails.overview.generalInformation.pullCommand')}
                  value={response.pullCommand}
                  withCodeBlock
                />
              </Container>
            </Layout.Vertical>
          </Card>
        </Layout.Vertical>
      )}
    </Page.Body>
  )
}
