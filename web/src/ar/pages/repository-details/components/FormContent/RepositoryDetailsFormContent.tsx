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
import { FormikContextType, connect } from 'formik'
import { Container, FormInput } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { Separator } from '@ar/components/Separator/Separator'
import { Description, Tags } from '@ar/components/NameDescriptionTags'
import type { VirtualRegistryRequest } from '@ar/pages/repository-details/types'

import RepositoryVisibilityContent from './RepositoryVisibilityContent'

interface RepositoryDetailsFormContentProps {
  isEdit: boolean
  readonly: boolean
}

function RepositoryDetailsFormContent(
  props: RepositoryDetailsFormContentProps & { formik: FormikContextType<VirtualRegistryRequest> }
): JSX.Element {
  const { formik, readonly, isEdit } = props
  const { getString } = useStrings()
  const { values } = formik
  const { description, labels } = values
  return (
    <Container data-testid="registry-definition">
      <FormInput.Text
        name="identifier"
        inputGroup={{
          autoFocus: true
        }}
        label={getString('repositoryDetails.repositoryForm.name')}
        placeholder={getString('repositoryDetails.repositoryForm.name')}
        disabled={readonly || isEdit}
      />
      {/* <EnvironmentSelect name="environment" /> */}
      <Description hasValue={!!description} disabled={readonly} />
      <Tags name="labels" hasValue={!isEmpty(labels)} disabled={readonly} />
      <Separator />
      <RepositoryVisibilityContent disabled={readonly} />
    </Container>
  )
}

export default connect<RepositoryDetailsFormContentProps, VirtualRegistryRequest>(RepositoryDetailsFormContent)
