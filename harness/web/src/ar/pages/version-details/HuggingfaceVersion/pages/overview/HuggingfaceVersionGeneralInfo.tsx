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
import moment from 'moment'
import { defaultTo } from 'lodash-es'
import { FontVariation } from '@harnessio/design-system'
import { Card, Container, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'
import { LabelValueTypeEnum } from '@ar/pages/version-details/components/LabelValueContent/type'
import { useVersionOverview } from '@ar/pages/version-details/context/VersionOverviewProvider'
import { LabelValueContent } from '@ar/pages/version-details/components/LabelValueContent/LabelValueContent'

import type { HuggingfaceArtifactDetails } from '../../types'
import css from './overview.module.scss'

interface HuggingfaceVersionGeneralInfoProps {
  className?: string
}

export default function HuggingfaceVersionGeneralInfo(props: HuggingfaceVersionGeneralInfoProps) {
  const { className } = props
  const { data } = useVersionOverview<HuggingfaceArtifactDetails>()
  const { getString } = useStrings()
  const lastModifiedInstance = data.metadata?.lastModified ? moment(data.metadata?.lastModified) : null
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
              label={getString('versionDetails.overview.generalInformation.name')}
              value={data.name}
              type={LabelValueTypeEnum.Text}
            />
            <LabelValueContent
              label={getString('versionDetails.overview.generalInformation.version')}
              value={data.version}
              type={LabelValueTypeEnum.Text}
            />
            <LabelValueContent
              label={getString('versionDetails.overview.generalInformation.artifactType')}
              value={data.artifactType}
              type={LabelValueTypeEnum.Text}
            />
            <LabelValueContent
              label={getString('versionDetails.overview.generalInformation.packageType')}
              value={getString('packageTypes.huggingfacePackage')}
              type={LabelValueTypeEnum.PackageType}
              icon="huggingface"
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
            {data.metadata?.projectUrl && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.projectUrl')}
                value={data.metadata.projectUrl}
                type={LabelValueTypeEnum.Text}
              />
            )}
            {data.metadata?.cardData?.license && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.license')}
                value={data.metadata.cardData.license}
                type={LabelValueTypeEnum.Text}
              />
            )}
            {Array.isArray(data.metadata?.cardData?.tags) && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.tags')}
                value={data.metadata?.cardData?.tags?.join(', ')}
                type={LabelValueTypeEnum.Text}
              />
            )}
            {Array.isArray(data.metadata?.cardData?.language) && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.language')}
                value={data.metadata?.cardData?.language?.join(', ')}
                type={LabelValueTypeEnum.Text}
              />
            )}
            {data.createdAt && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.createdAt')}
                value={getReadableDateTime(Number(data.createdAt), DEFAULT_DATE_TIME_FORMAT)}
                type={LabelValueTypeEnum.Text}
              />
            )}
            {lastModifiedInstance && lastModifiedInstance.isValid() && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.modifiedAt')}
                value={getReadableDateTime(lastModifiedInstance.valueOf(), DEFAULT_DATE_TIME_FORMAT)}
                type={LabelValueTypeEnum.Text}
              />
            )}
          </Container>
        </Layout.Horizontal>
      </Layout.Vertical>
    </Card>
  )
}
