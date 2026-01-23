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
import { Container, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import type { ArtifactScanDetails } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'

import InformationMetrics from './InformationMetrics'

import css from './ViolationDetailsContent.module.scss'

interface EvaluationInformationContentProps {
  data: ArtifactScanDetails
}

function EvaluationInformationContent({ data }: EvaluationInformationContentProps) {
  const { getString } = useStrings()
  return (
    <Layout.Vertical spacing="large">
      <Text font={{ variation: FontVariation.H5, weight: 'bold' }} color={Color.GREY_700}>
        {getString('violationsList.violationDetailsModal.evaluationDetailsSection.title')}
      </Text>
      <Container className={css.gridContainer}>
        <InformationMetrics.Text
          label={getString('violationsList.violationDetailsModal.evaluationDetailsSection.firstDetected')}
          value={getReadableDateTime(Number(data.lastEvaluatedAt), DEFAULT_DATE_TIME_FORMAT) || getString('na')}
        />
        <InformationMetrics.Text
          label={getString('violationsList.violationDetailsModal.evaluationDetailsSection.lastDetected')}
          value={getReadableDateTime(Number(data.lastEvaluatedAt), DEFAULT_DATE_TIME_FORMAT) || getString('na')}
        />
      </Container>
    </Layout.Vertical>
  )
}

export default EvaluationInformationContent
