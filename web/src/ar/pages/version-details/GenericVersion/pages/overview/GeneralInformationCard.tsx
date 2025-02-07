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
import { isEmpty } from 'lodash-es'
import { FontVariation } from '@harnessio/design-system'
import { Card, Container, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'
import { LabelValueContent } from '@ar/pages/version-details/components/LabelValueContent/LabelValueContent'

import type { GenericArtifactDetails } from '../../types'

import css from './styles.module.scss'

interface GeneralInformationCardProps {
  data: GenericArtifactDetails
  className?: string
}

export default function GeneralInformationCard(props: GeneralInformationCardProps) {
  const { data, className } = props
  const { getString } = useStrings()
  return (
    <Card className={className} title={getString('versionDetails.overview.generalInformation.title')}>
      <Layout.Vertical spacing="medium">
        <Text font={{ variation: FontVariation.CARD_TITLE }}>
          {getString('versionDetails.overview.generalInformation.title')}
        </Text>
        <Container className={css.gridContainer}>
          <LabelValueContent
            label={getString('versionDetails.overview.generalInformation.name')}
            value={data.name}
            withCopyText
          />
          <LabelValueContent
            label={getString('versionDetails.overview.generalInformation.version')}
            value={data.version}
            withCopyText
          />
          <Text font={{ variation: FontVariation.SMALL_BOLD }}>
            {getString('versionDetails.overview.generalInformation.packageType')}
          </Text>
          <Text icon="generic-repository-type" iconProps={{ size: 20 }} font={{ variation: FontVariation.SMALL }}>
            {getString('packageTypes.genericPackage')}
          </Text>
          <LabelValueContent
            label={getString('versionDetails.overview.generalInformation.uploadedBy')}
            value={getReadableDateTime(Number(data.modifiedAt), DEFAULT_DATE_TIME_FORMAT)}
          />
          <LabelValueContent
            label={getString('versionDetails.overview.generalInformation.description')}
            value={isEmpty(data.description) ? getString('na') : data.description}
          />
        </Container>
      </Layout.Vertical>
    </Card>
  )
}
