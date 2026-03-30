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
import { Container, FormInput, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

interface ExemptionDetailsAndJustificationFormContentProps {
  isEdit?: boolean
}

export default function ExemptionDetailsAndJustificationFormContent(
  _props: ExemptionDetailsAndJustificationFormContentProps
) {
  const { getString } = useStrings()
  return (
    <Layout.Vertical spacing="medium">
      <Text font={{ variation: FontVariation.CARD_TITLE }}>
        {getString('violationsList.createExemptionForm.exemptionDetailsAndJustificationSection.title')}
      </Text>
      <Container>
        <FormInput.Text
          name="expireAfter"
          label={getString(
            'violationsList.createExemptionForm.exemptionDetailsAndJustificationSection.exemptionDuration'
          )}
          placeholder={getString(
            'violationsList.createExemptionForm.exemptionDetailsAndJustificationSection.exemptionDuration'
          )}
        />
        <FormInput.TextArea
          label={getString(
            'violationsList.createExemptionForm.exemptionDetailsAndJustificationSection.businessJustification'
          )}
          textArea={{ style: { minHeight: 130 } }}
          name="businessJustification"
          placeholder={getString(
            'violationsList.createExemptionForm.exemptionDetailsAndJustificationSection.businessJustificationPlaceholder'
          )}
        />
        <FormInput.TextArea
          label={getString(
            'violationsList.createExemptionForm.exemptionDetailsAndJustificationSection.remediationPlan'
          )}
          name="remediationPlan"
          textArea={{ style: { minHeight: 130 } }}
          placeholder={getString(
            'violationsList.createExemptionForm.exemptionDetailsAndJustificationSection.remediationPlanPlaceholder'
          )}
        />
      </Container>
    </Layout.Vertical>
  )
}
