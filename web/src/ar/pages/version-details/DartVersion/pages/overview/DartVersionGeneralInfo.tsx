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
import { Card, Container, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { LabelValueTypeEnum } from '@ar/pages/version-details/components/LabelValueContent/type'
import { useVersionOverview } from '@ar/pages/version-details/context/VersionOverviewProvider'
import { LabelValueContent } from '@ar/pages/version-details/components/LabelValueContent/LabelValueContent'

import type { DartArtifactDetails } from '../../types'
import css from './overview.module.scss'

interface DartVersionGeneralInfoProps {
  className?: string
}

export default function DartVersionGeneralInfo(props: DartVersionGeneralInfoProps) {
  const { className } = props
  const { data } = useVersionOverview<DartArtifactDetails>()
  const { getString } = useStrings()
  const metadata = data.metadata
  const pubspec = metadata?.pubspec
  return (
    <Card
      data-testid="general-information-card"
      className={className}
      title={getString('versionDetails.overview.generalInformation.title')}>
      <Layout.Vertical spacing="medium">
        <Text font={{ variation: FontVariation.CARD_TITLE }}>
          {getString('versionDetails.overview.generalInformation.title')}
        </Text>
        <Layout.Horizontal
          margin={{ top: 'medium' }}
          spacing="xlarge"
          flex={{ alignItems: 'flex-start', justifyContent: 'flex-start' }}>
          <Container className={css.gridContainer}>
            <LabelValueContent
              label={getString('versionDetails.overview.generalInformation.packageType')}
              value={getString('packageTypes.dartPackage')}
              type={LabelValueTypeEnum.PackageType}
              icon="dart-icon"
            />
            <LabelValueContent
              label={getString('versionDetails.overview.generalInformation.size')}
              value={data.size}
              type={LabelValueTypeEnum.Text}
            />
            <LabelValueContent
              label={getString('versionDetails.overview.generalInformation.downloads')}
              value={defaultTo(data.downloadCount?.toLocaleString(), 0)}
              type={LabelValueTypeEnum.Text}
            />
            {metadata?.documentation_url && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.documentation')}
                value={metadata.documentation_url}
                type={LabelValueTypeEnum.Link}
              />
            )}
            {metadata?.project_url && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.projectUrl')}
                value={metadata.project_url}
                type={LabelValueTypeEnum.Link}
              />
            )}
            {metadata?.repository_url && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.repository')}
                value={metadata.repository_url}
                type={LabelValueTypeEnum.Link}
              />
            )}
            {pubspec?.home_page && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.homepage')}
                value={pubspec.home_page}
                type={LabelValueTypeEnum.Link}
              />
            )}
            {metadata?.description && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.description')}
                value={metadata.description}
                type={LabelValueTypeEnum.Text}
              />
            )}
          </Container>
        </Layout.Horizontal>
      </Layout.Vertical>
    </Card>
  )
}
