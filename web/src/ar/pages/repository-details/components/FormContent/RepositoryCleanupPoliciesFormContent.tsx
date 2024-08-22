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
import { uniqueId } from 'lodash-es'
import { FieldArray, type FormikContextType, connect } from 'formik'
import { Color, FontVariation } from '@harnessio/design-system'
import { Container, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import CleanupPolicyList from '@ar/components/CleanupPolicyList/CleanupPolicyList'
import type { VirtualRegistryRequest } from '@ar/pages/repository-details/types'

interface RepositoryCleanupPoliciesFormContentProps {
  isEdit: boolean
  disabled: boolean
}

function RepositoryCleanupPoliciesFormContent(
  props: RepositoryCleanupPoliciesFormContentProps & { formik: FormikContextType<VirtualRegistryRequest> }
): JSX.Element {
  const { formik, disabled } = props
  const { getString } = useStrings()

  const getDefaultValueToAdd = () => {
    return {
      name: '',
      id: uniqueId('cleanupPolicies')
    }
  }

  return (
    <Layout.Vertical flex={{ alignItems: 'flex-start' }} spacing="xsmall">
      <Text font={{ variation: FontVariation.CARD_TITLE }}>
        {getString('repositoryDetails.repositoryForm.cleanupPoliciesTitle')}
      </Text>
      <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
        {getString('repositoryDetails.repositoryForm.cleanupPoliciesSubTitle')}
      </Text>
      <Container width="100%">
        <FieldArray
          name="cleanupPolicy"
          render={({ push, remove }) => {
            return (
              <CleanupPolicyList
                addButtonLabel={getString('cleanupPolicy.addBtn')}
                onAdd={push}
                onRemove={remove}
                name="cleanupPolicy"
                formikProps={formik}
                disabled={disabled}
                getDefaultValue={getDefaultValueToAdd}
              />
            )
          }}
        />
      </Container>
    </Layout.Vertical>
  )
}

export default connect<RepositoryCleanupPoliciesFormContentProps, VirtualRegistryRequest>(
  RepositoryCleanupPoliciesFormContent
)
