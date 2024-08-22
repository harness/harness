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
import type { FormikProps } from 'formik'
import { FormInput, Layout } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings/String'
import RepositoryUrlInput from '../RepositoryUrlInput/RepositoryUrlInput'
import type { UpstreamRegistryRequest } from '../../types'

interface UpstreamProxyDetailsFormContentProps {
  readonly: boolean
  formikProps: FormikProps<UpstreamRegistryRequest>
  isEdit: boolean
}

export default function UpstreamProxyDetailsFormContent(props: UpstreamProxyDetailsFormContentProps): JSX.Element {
  const { readonly, isEdit, formikProps } = props
  const { values } = formikProps
  const { packageType } = values
  const { getString } = useStrings()
  return (
    <Layout.Vertical>
      <FormInput.Text
        name="identifier"
        label={getString('upstreamProxyDetails.createForm.key')}
        placeholder={getString('upstreamProxyDetails.createForm.key')}
        disabled={readonly || isEdit}
        inputGroup={{
          autoFocus: true
        }}
      />
      <RepositoryUrlInput packageType={packageType} readonly={readonly} />
    </Layout.Vertical>
  )
}
