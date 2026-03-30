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
import type { ArtifactScanV3 } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { Separator } from '@ar/components/Separator/Separator'
import BasicInformationFormContent from './BasicInformationFormContent'
import ExemptionDetailsAndJustificationFormContent from './ExemptionDetailsAndJustificationFormContent'

interface CreateExemptionFormContentProps {
  data: ArtifactScanV3
  isEdit?: boolean
  title?: string
  subTitle?: string
}

function CreateExemptionFormContent(props: CreateExemptionFormContentProps) {
  const { getString } = useStrings()
  return (
    <Container>
      <Layout.Vertical spacing="medium">
        <Text font={{ variation: FontVariation.H3 }}>
          {props.title || getString('violationsList.createExemptionForm.title')}
        </Text>
        <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_600}>
          {props.subTitle || getString('violationsList.createExemptionForm.subTitle')}
        </Text>
      </Layout.Vertical>
      <Separator />
      <BasicInformationFormContent data={props.data} isEdit={props.isEdit} />
      <ExemptionDetailsAndJustificationFormContent isEdit={props.isEdit} />
    </Container>
  )
}

export default CreateExemptionFormContent
