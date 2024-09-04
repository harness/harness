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
import { FontVariation } from '@harnessio/design-system'
import { Card, Container, Layout, Page, Text } from '@harnessio/uicore'
import { useGetDockerArtifactDetailsQuery } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import { getReadableDateTime } from '@ar/common/dateUtils'
import { useDecodedParams, useGetSpaceRef, useParentHooks } from '@ar/hooks'

import type { DockerVersionDetailsQueryParams } from './types'
import DockerVersionOverviewCards from './components/OverviewCards/OverviewCards'
import { LabelValueContent } from '../components/LabelValueContent/LabelValueContent'

import css from './DockerVersion.module.scss'

export default function DockerVersionOverviewContent(): JSX.Element {
  const { getString } = useStrings()
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const { useQueryParams } = useParentHooks()
  const { digest } = useQueryParams<DockerVersionDetailsQueryParams>()
  const spaceRef = useGetSpaceRef()

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetDockerArtifactDetailsQuery(
    {
      registry_ref: spaceRef,
      artifact: encodeRef(pathParams.artifactIdentifier),
      version: pathParams.versionIdentifier,
      queryParams: {
        digest
      }
    },
    {
      enabled: !!digest
    }
  )

  const response = data?.content?.data

  return (
    <Page.Body
      className={css.pageBody}
      loading={loading || !digest}
      error={error?.message}
      retryOnError={() => refetch()}>
      {response && (
        <Layout.Vertical className={css.cardContainer} spacing="medium" flex={{ alignItems: 'flex-start' }}>
          <DockerVersionOverviewCards />
          <Card title="General Information">
            <Layout.Vertical spacing="medium">
              <Text font={{ variation: FontVariation.CARD_TITLE }}>
                {getString('versionDetails.overview.generalInformation.title')}
              </Text>
              <Container className={css.gridContainer}>
                <LabelValueContent
                  label={getString('versionDetails.overview.generalInformation.name')}
                  value={response.imageName}
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
                <Text icon="docker-step" iconProps={{ size: 16 }} font={{ variation: FontVariation.SMALL }}>
                  {getString('packageTypes.dockerPackage')}
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
                  value={response.downloadsCount?.toString()}
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
