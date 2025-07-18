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

import type { GoArtifactDetails } from '../../types'
import css from './overview.module.scss'

interface GoVersionGeneralInfoProps {
  className?: string
}

export default function GoVersionGeneralInfo(props: GoVersionGeneralInfoProps) {
  const { className } = props
  const { data } = useVersionOverview<GoArtifactDetails>()
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
              value={getString('packageTypes.goPackage')}
              type={LabelValueTypeEnum.PackageType}
              icon="go-logo"
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
            {data.metadata?.Origin?.VCS && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.vcs')}
                value={data.metadata.Origin.VCS}
                type={LabelValueTypeEnum.Text}
              />
            )}
            {data.metadata?.Origin?.URL && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.url')}
                value={data.metadata.Origin.URL}
                type={LabelValueTypeEnum.Text}
              />
            )}
            {data.metadata?.Origin?.Ref && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.ref')}
                value={data.metadata.Origin.Ref}
                type={LabelValueTypeEnum.Text}
              />
            )}
            {data.metadata?.Origin?.Hash && (
              <LabelValueContent
                label={getString('versionDetails.overview.generalInformation.hash')}
                value={data.metadata.Origin.Hash}
                type={LabelValueTypeEnum.Text}
              />
            )}
          </Container>
        </Layout.Horizontal>
      </Layout.Vertical>
    </Card>
  )
}
