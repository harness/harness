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
import { FormikContextType, connect } from 'formik'
import { FormInput, Layout } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings/String'
import { DockerRepositoryURLInputSource, UpstreamRegistryRequest } from '@ar/pages/upstream-proxy-details/types'

interface DockerRepositoryUrlInputProps {
  readonly: boolean
}

function DockerRepositoryUrlInput(
  props: DockerRepositoryUrlInputProps & { formik: FormikContextType<UpstreamRegistryRequest> }
): JSX.Element {
  const { readonly, formik } = props
  const { values } = formik
  const { getString } = useStrings()
  const { config } = values
  const { source } = config
  return (
    <Layout.Vertical spacing="small">
      <FormInput.RadioGroup
        name="config.source"
        radioGroup={{ inline: true }}
        disabled={readonly}
        label={getString('upstreamProxyDetails.createForm.source.title')}
        items={[
          {
            label: getString('upstreamProxyDetails.createForm.source.dockerHub'),
            value: DockerRepositoryURLInputSource.Dockerhub
          },
          {
            label: getString('upstreamProxyDetails.createForm.source.custom'),
            value: DockerRepositoryURLInputSource.Custom
          }
        ]}
      />
      {source === DockerRepositoryURLInputSource.Custom && (
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

export default connect<DockerRepositoryUrlInputProps, UpstreamRegistryRequest>(DockerRepositoryUrlInput)
