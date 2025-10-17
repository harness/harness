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
import { useFormikContext } from 'formik'
import { FormInput, Layout } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings/String'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'

import { UpstreamURLSourceConfig } from './constants'
import { UpstreamProxyAuthenticationMode, UpstreamRegistryRequest, UpstreamRepositoryURLInputSource } from '../../types'

interface RepositoryUrlInputProps {
  readonly: boolean
}

export default function RepositoryUrlInput(props: RepositoryUrlInputProps): JSX.Element {
  const { getString } = useStrings()
  const { readonly } = props
  const { values, setFieldValue } = useFormikContext<UpstreamRegistryRequest>()
  const { packageType, config } = values
  const { source } = config
  const repositoryType = repositoryFactory.getRepositoryType(packageType)
  const supportedURLSources = repositoryType?.getSupportedUpstreamURLSources() || []
  const radioItems = supportedURLSources.map(each => UpstreamURLSourceConfig[each])
  return (
    <Layout.Vertical spacing="small">
      {radioItems.length ? (
        <FormInput.RadioGroup
          key={packageType}
          name="config.source"
          radioGroup={{ inline: true }}
          disabled={readonly}
          label={getString('upstreamProxyDetails.createForm.source.title')}
          items={radioItems.map(each => ({ ...each, label: getString(each.label) }))}
          onChange={e => {
            const selectedValue = e.currentTarget.value as UpstreamRepositoryURLInputSource
            if (source !== selectedValue) {
              setFieldValue('config.url', '')
              setFieldValue('config.authType', UpstreamProxyAuthenticationMode.ANONYMOUS)
            }
          }}
        />
      ) : null}
      {[UpstreamRepositoryURLInputSource.Custom, UpstreamRepositoryURLInputSource.AwsEcr].includes(
        source as UpstreamRepositoryURLInputSource
      ) && (
        <FormInput.Text
          name="config.url"
          label={getString('upstreamProxyDetails.createForm.url')}
          placeholder={getString('upstreamProxyDetails.createForm.url')}
          disabled={readonly}
        />
      )}
    </Layout.Vertical>
  )
}
