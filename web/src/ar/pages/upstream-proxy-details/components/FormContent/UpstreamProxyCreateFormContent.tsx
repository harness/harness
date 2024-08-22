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
import { FormikProps, connect } from 'formik'
import { FontVariation } from '@harnessio/design-system'
import { FormikForm, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings/String'
import UpstreamProxyDetailsFormContent from './UpstreamProxyDetailsFormContent'
import UpstreamProxyAuthenticationFormContent from './UpstreamProxyAuthenticationFormContent'
import type { UpstreamRegistryRequest } from '../../types'

interface UpstreamProxyFormContentProps {
  readonly: boolean
  isEdit: boolean
}

function UpstreamProxyCreateFormContent(
  props: UpstreamProxyFormContentProps & { formik: FormikProps<UpstreamRegistryRequest> }
): JSX.Element {
  const { readonly, formik, isEdit } = props
  const { getString } = useStrings()
  return (
    <FormikForm>
      <Layout.Vertical>
        <Text font={{ variation: FontVariation.CARD_TITLE }} margin={{ bottom: 'medium' }}>
          {getString('upstreamProxyDetails.form.title')}
        </Text>
        <UpstreamProxyDetailsFormContent isEdit={isEdit} formikProps={formik} readonly={readonly} />
        <UpstreamProxyAuthenticationFormContent formikProps={formik} readonly={readonly} />
      </Layout.Vertical>
    </FormikForm>
  )
}

export default connect<UpstreamProxyFormContentProps, UpstreamRegistryRequest>(UpstreamProxyCreateFormContent)
