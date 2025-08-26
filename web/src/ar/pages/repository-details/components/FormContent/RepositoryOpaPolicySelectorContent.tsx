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
import { get } from 'lodash-es'
import { Container, Layout, Text } from '@harnessio/uicore'
import { connect, type FormikContextType } from 'formik'
import { Color, FontVariation } from '@harnessio/design-system'

import { useParentComponents } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { VirtualRegistryRequest } from '@ar/pages/repository-details/types'

import { POLICY_ACTION, POLICY_TYPE } from '../../constants'

interface RepositoryOpaPolicySelectorContentProps {
  disabled: boolean
}

function RepositoryOpaPolicySelectorContent(
  props: RepositoryOpaPolicySelectorContentProps & { formik: FormikContextType<VirtualRegistryRequest> }
): JSX.Element {
  const { disabled, formik } = props
  const { getString } = useStrings()
  const { PolicySetFixedTypeSelector } = useParentComponents()
  const { values } = formik
  const policies = get(values, 'policyRefs', []) || []
  return (
    <Layout.Vertical spacing="xsmall">
      <Text font={{ variation: FontVariation.H6 }}>
        {getString('repositoryDetails.repositoryForm.opaPolicy.title')}
      </Text>
      <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
        {getString('repositoryDetails.repositoryForm.opaPolicy.subTitle')}
      </Text>
      {PolicySetFixedTypeSelector && (
        <Container padding={{ top: 'small' }}>
          <PolicySetFixedTypeSelector
            formik={formik}
            name="policyRefs"
            policySetIds={policies}
            policyType={POLICY_TYPE}
            policyAction={POLICY_ACTION}
            disabled={disabled}
            buttonProps={{
              text: getString('repositoryDetails.repositoryForm.opaPolicy.addTitle'),
              icon: 'plus',
              font: { variation: FontVariation.FORM_LABEL },
              withoutCurrentColor: false
            }}
          />
        </Container>
      )}
    </Layout.Vertical>
  )
}

export default connect<RepositoryOpaPolicySelectorContentProps, VirtualRegistryRequest>(
  RepositoryOpaPolicySelectorContent
)
