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
import { connect, type FormikContextType } from 'formik'
import { FormikForm, Layout, Text } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'

import { useStrings } from '@ar/frameworks/strings'
import type { VirtualRegistryRequest } from '@ar/pages/repository-details/types'
import RepositoryDetailsFormContent from './RepositoryDetailsFormContent'

interface RepositoryCreateFormContentProps {
  isEdit: boolean
}

function RepositoryCreateFormContent(
  props: RepositoryCreateFormContentProps & { formik: FormikContextType<VirtualRegistryRequest> }
): JSX.Element {
  const { isEdit } = props
  const { getString } = useStrings()
  return (
    <FormikForm>
      <Layout.Vertical>
        <Text font={{ variation: FontVariation.CARD_TITLE }} margin={{ bottom: 'medium' }}>
          {getString('repositoryDetails.repositoryForm.title')}
        </Text>
        <RepositoryDetailsFormContent isEdit={isEdit} readonly={false} />
      </Layout.Vertical>
    </FormikForm>
  )
}

export default connect<RepositoryCreateFormContentProps, VirtualRegistryRequest>(RepositoryCreateFormContent)
