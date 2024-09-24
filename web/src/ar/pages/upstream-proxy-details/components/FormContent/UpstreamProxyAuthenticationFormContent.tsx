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
import type { FormikProps } from 'formik'
import { FontVariation } from '@harnessio/design-system'
import { Container, FormInput, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { useAppStore, useParentComponents } from '@ar/hooks'
import { UpstreamProxyAuthenticationMode, type UpstreamRegistryRequest } from '../../types'
import css from './FormContent.module.scss'

interface UpstreamProxyAuthenticationFormContentProps {
  formikProps: FormikProps<UpstreamRegistryRequest>
  readonly: boolean
}

export default function UpstreamProxyAuthenticationFormContent({
  formikProps,
  readonly
}: UpstreamProxyAuthenticationFormContentProps): JSX.Element {
  const { getString } = useStrings()
  const { scope } = useAppStore()
  const { SecretFormInput } = useParentComponents()
  const selectedRadioValue = get(formikProps.values, 'config.authType')
  return (
    <Layout.Vertical spacing="small">
      <FormInput.RadioGroup
        name="config.authType"
        onChange={e => {
          const selectedValue = e.currentTarget.value as UpstreamProxyAuthenticationMode
          if (selectedValue === UpstreamProxyAuthenticationMode.ANONYMOUS) {
            formikProps.setFieldValue('config.auth', null)
          } else {
            formikProps.setFieldValue('config.auth', {})
          }
        }}
        radioGroup={{ inline: true }}
        label={getString('upstreamProxyDetails.createForm.authentication.title')}
        items={[
          {
            label: getString('upstreamProxyDetails.createForm.authentication.userNameAndPassword'),
            value: UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD
          },
          {
            label: (
              <Layout.Vertical>
                <Text font={{ variation: FontVariation.SMALL }}>
                  {getString('upstreamProxyDetails.createForm.authentication.anonymous')}
                </Text>
                <Text font={{ variation: FontVariation.SMALL }}>
                  {getString('upstreamProxyDetails.createForm.authentication.anonymousSubLabel')}
                </Text>
              </Layout.Vertical>
            ),
            value: UpstreamProxyAuthenticationMode.ANONYMOUS
          }
        ]}
        disabled={readonly}
      />
      {selectedRadioValue === UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD && (
        <Container className={css.authContainer}>
          <Layout.Vertical>
            <FormInput.Text
              name="config.auth.userName"
              label={getString('upstreamProxyDetails.createForm.authentication.username')}
              placeholder={getString('upstreamProxyDetails.createForm.authentication.username')}
              disabled={readonly}
            />
            {/* <FormInput.Text
              name="config.auth.password"
              label={getString('upstreamProxyDetails.createForm.authentication.password')}
              placeholder={getString('upstreamProxyDetails.createForm.authentication.password')}
              disabled={readonly}
              inputGroup={{
                type: 'password'
              }}
            /> */}
            <SecretFormInput
              name="config.auth.secretIdentifier"
              spaceIdFieldName="config.auth.secretSpaceId"
              label={getString('upstreamProxyDetails.createForm.authentication.password')}
              placeholder={getString('upstreamProxyDetails.createForm.authentication.password')}
              scope={scope}
              disabled={readonly}
              formik={formikProps}
            />
          </Layout.Vertical>
        </Container>
      )}
    </Layout.Vertical>
  )
}
