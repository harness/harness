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

import type { NpmVersionDetailsConfig } from '../../types'
import css from './overview.module.scss'

interface NpmVersionGeneralInfoProps {
  className?: string
}

export default function NpmVersionGeneralInfo(props: NpmVersionGeneralInfoProps) {
  const { className } = props
  const { data } = useVersionOverview<NpmVersionDetailsConfig>()
  const versionMetatdata = data.metadata?.versions?.[data.version]
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
        <Container className={css.gridContainer}>
          <LabelValueContent
            label={getString('versionDetails.overview.generalInformation.packageType')}
            value={getString('packageTypes.npmPackage')}
            type={LabelValueTypeEnum.PackageType}
            icon="npm-repository-type"
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
          {versionMetatdata?.repository && (
            <LabelValueContent
              label={getString('versionDetails.overview.generalInformation.repository')}
              value={versionMetatdata.repository?.url}
              type={LabelValueTypeEnum.Link}
            />
          )}
          {versionMetatdata?.homepage && (
            <LabelValueContent
              label={getString('versionDetails.overview.generalInformation.homepage')}
              value={versionMetatdata.homepage}
              type={LabelValueTypeEnum.Link}
            />
          )}
          {versionMetatdata?.license && (
            <LabelValueContent
              label={getString('versionDetails.overview.generalInformation.license')}
              value={versionMetatdata.license}
              type={LabelValueTypeEnum.Text}
            />
          )}

          <LabelValueContent
            label={getString('versionDetails.overview.generalInformation.uploadedBy')}
            value={getReadableDateTime(Number(data.modifiedAt), DEFAULT_DATE_TIME_FORMAT)}
            type={LabelValueTypeEnum.Text}
          />
          {data.metadata?.pullCommand && (
            <LabelValueContent
              label={getString('versionDetails.overview.generalInformation.pullCommand')}
              value={data.metadata.pullCommand}
              type={LabelValueTypeEnum.CommandBlock}
            />
          )}
        </Container>
      </Layout.Vertical>
    </Card>
  )
}
