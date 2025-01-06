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
import { useFormikContext } from 'formik'
import { FontVariation } from '@harnessio/design-system'
import { Container, FormInput, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { useAppStore, useParentComponents } from '@ar/hooks'
import MultiTypeSecretInput from '@ar/components/MultiTypeSecretInput/MultiTypeSecretInput'

import { AuthTypeRadioItems, URLSourceToSupportedAuthTypesMapping } from './constants'
import { UpstreamProxyAuthenticationMode, UpstreamRegistryRequest, UpstreamRepositoryURLInputSource } from '../../types'

import css from './AuthenticationFormInput.module.scss'

interface AuthenticationFormInputProps {
  readonly?: boolean
}

export default function AuthenticationFormInput(props: AuthenticationFormInputProps): JSX.Element {
  const { readonly } = props
  const { scope } = useAppStore()
  const { getString } = useStrings()
  const { SecretFormInput } = useParentComponents()
  const formikProps = useFormikContext<UpstreamRegistryRequest>()

  const { values, setFieldValue } = formikProps
  const { config, packageType } = values
  const { source } = config
  const selectedRadioValue = get(values, 'config.authType')
  const supportedAuthTypes = URLSourceToSupportedAuthTypesMapping[source as UpstreamRepositoryURLInputSource] || []
  const radioItems = supportedAuthTypes.map(each => AuthTypeRadioItems[each])
  return (
    <Layout.Vertical spacing="small">
      <FormInput.RadioGroup
        key={`${source}-${packageType}`}
        name="config.authType"
        onChange={e => {
          const selectedValue = e.currentTarget.value as UpstreamProxyAuthenticationMode
          if (selectedValue === UpstreamProxyAuthenticationMode.ANONYMOUS) {
            setFieldValue('config.auth', null)
          } else {
            setFieldValue('config.auth', {})
          }
        }}
        radioGroup={{ inline: true }}
        label={getString('upstreamProxyDetails.createForm.authentication.title')}
        items={radioItems.map(each => ({
          ...each,
          label: (
            <Layout.Vertical>
              <Text font={{ variation: FontVariation.SMALL }}>{getString(each.label)}</Text>
              {each.subLabel && <Text font={{ variation: FontVariation.SMALL }}>{getString(each.subLabel)}</Text>}
            </Layout.Vertical>
          )
        }))}
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
      {selectedRadioValue === UpstreamProxyAuthenticationMode.ACCESS_KEY_AND_SECRET_KEY && (
        <Container className={css.authContainer}>
          <Layout.Vertical>
            <MultiTypeSecretInput
              className={css.multiTypeSecretInput}
              labelClassName={css.labelContainer}
              name="config.auth.accessKey"
              secretField="config.auth.accessKeySecretIdentifier"
              secretSpaceIdField="config.auth.accessKeySecretSpaceId"
              typeField="config.auth.accessKeyType"
              label={getString('upstreamProxyDetails.createForm.authentication.accessKey')}
              placeholder={getString('upstreamProxyDetails.createForm.authentication.accessKey')}
              disabled={readonly}
            />
            <SecretFormInput
              name="config.auth.secretKeyIdentifier"
              spaceIdFieldName="config.auth.secretKeySpaceId"
              label={getString('upstreamProxyDetails.createForm.authentication.secretKey')}
              placeholder={getString('upstreamProxyDetails.createForm.authentication.secretKey')}
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
