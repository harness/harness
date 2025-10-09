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
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'
import { LabelValueTypeEnum } from '@ar/pages/version-details/components/LabelValueContent/type'
import { useVersionOverview } from '@ar/pages/version-details/context/VersionOverviewProvider'
import { LabelValueContent } from '@ar/pages/version-details/components/LabelValueContent/LabelValueContent'

import type { RPMArtifactDetails } from '../../types'
import css from './overview.module.scss'

interface RPMVersionGeneralInfoProps {
  className?: string
}

export default function RPMVersionGeneralInfo(props: RPMVersionGeneralInfoProps) {
  const { className } = props
  const { data } = useVersionOverview<RPMArtifactDetails>()
  const { getString } = useStrings()
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
              value={getString('packageTypes.rpmPackage')}
              type={LabelValueTypeEnum.PackageType}
              icon="red-hat-logo"
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
            {data.metadata?.version_metadata?.project_url && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.homepage')}
                value={data.metadata.version_metadata.project_url}
                type={LabelValueTypeEnum.Link}
              />
            )}
            {data.metadata?.version_metadata?.license && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.license')}
                value={data.metadata.version_metadata.license}
                type={LabelValueTypeEnum.Text}
              />
            )}
            <LabelValueContent
              label={getString('versionDetails.overview.generalInformation.uploadedBy')}
              value={getReadableDateTime(Number(data.modifiedAt), DEFAULT_DATE_TIME_FORMAT)}
              type={LabelValueTypeEnum.Text}
            />
            {data.metadata?.version_metadata?.summary && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.description')}
                value={data.metadata.version_metadata.summary}
                type={LabelValueTypeEnum.Text}
                lineClamp={3}
              />
            )}
          </Container>
          <Container className={css.gridContainer}>
            {data.metadata?.file_metadata?.build_host && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.buildHost')}
                value={data.metadata.file_metadata.build_host}
                type={LabelValueTypeEnum.Text}
              />
            )}
            {data.metadata?.file_metadata?.build_time && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.buildTime')}
                value={getReadableDateTime(data.metadata.file_metadata.build_time * 1000, DEFAULT_DATE_TIME_FORMAT)} // seconds to milliseconds
                type={LabelValueTypeEnum.Text}
              />
            )}
            {data.metadata?.file_metadata?.packager && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.packager')}
                value={data.metadata.file_metadata.packager}
                type={LabelValueTypeEnum.Text}
              />
            )}
            {data.metadata?.file_metadata?.architecture && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.platform')}
                value={data.metadata.file_metadata.architecture}
                type={LabelValueTypeEnum.Text}
              />
            )}
            {data.metadata?.file_metadata?.source_rpm && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.sourceRpm')}
                value={data.metadata.file_metadata.source_rpm}
                type={LabelValueTypeEnum.Text}
              />
            )}
            {data.metadata?.file_metadata?.vendor && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.vendor')}
                value={data.metadata.file_metadata.vendor}
                type={LabelValueTypeEnum.Text}
              />
            )}
          </Container>
        </Layout.Horizontal>
      </Layout.Vertical>
    </Card>
  )
}
