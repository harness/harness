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
import type { VersionDetailsPathParams } from '@ar/routes/types'
import { useDecodedParams, useGetSpaceRef } from '@ar/hooks'

import useGetOCIVersionParams from '../../hooks/useGetOCIVersionParams'
import { LabelValueTypeEnum } from '../../components/LabelValueContent/type'
import { LabelValueContent } from '../../components/LabelValueContent/LabelValueContent'

import css from '../DockerVersion.module.scss'

interface DockerVersionOSSGeneralInfoProps {
  className?: string
}

export default function DockerVersionOSSGeneralInfo({ className }: DockerVersionOSSGeneralInfoProps): JSX.Element {
  const { getString } = useStrings()
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const spaceRef = useGetSpaceRef()
  const { versionIdentifier, versionType, digest } = useGetOCIVersionParams()

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetDockerArtifactDetailsQuery(
    {
      registry_ref: spaceRef,
      artifact: encodeRef(pathParams.artifactIdentifier),
      version: versionIdentifier,
      queryParams: {
        digest,
        version_type: versionType
      }
    },
    {
      enabled: !!digest
    }
  )

  const response = data?.content?.data

  return (
    <Page.Body className={className} loading={loading || !digest} error={error?.message} retryOnError={() => refetch()}>
      {response && (
        <Card data-testid="general-information-card" className={css.cardContainer} title="General Information">
          <Layout.Vertical spacing="medium">
            <Text font={{ variation: FontVariation.CARD_TITLE }}>
              {getString('versionDetails.overview.generalInformation.title')}
            </Text>
            <Container className={css.gridContainer}>
              <LabelValueContent
                icon="docker-step"
                label={getString('versionDetails.overview.generalInformation.packageType')}
                value={getString('packageTypes.dockerPackage')}
                type={LabelValueTypeEnum.PackageType}
              />
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.repositoryPath')}
                value={response.registryPath}
                type={LabelValueTypeEnum.CopyText}
              />
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.url')}
                value={response.url}
                type={LabelValueTypeEnum.CopyText}
              />
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.pullCommand')}
                value={response.pullCommand}
                type={LabelValueTypeEnum.CommandBlock}
              />
            </Container>
          </Layout.Vertical>
        </Card>
      )}
    </Page.Body>
  )
}
