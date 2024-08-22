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
import { FormInput } from '@harnessio/uicore'
import type { PackageType } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings/String'
import { UpstreamProxyPackageType } from '@ar/pages/upstream-proxy-details/types'
import DockerRepositoryUrlInput from '@ar/pages/upstream-proxy-details/DockerRepository/DockerRepositoryUrlInput/DockerRepositoryUrlInput'

interface RepositoryUrlInputProps {
  readonly: boolean
  packageType: PackageType
}

export default function RepositoryUrlInput(props: RepositoryUrlInputProps): JSX.Element {
  const { getString } = useStrings()
  const { readonly, packageType } = props
  if (packageType === UpstreamProxyPackageType.DOCKER) {
    return <DockerRepositoryUrlInput readonly={readonly} />
  }
  return (
    <FormInput.Text
      name="config.url"
      label={getString('upstreamProxyDetails.createForm.url')}
      placeholder={getString('upstreamProxyDetails.createForm.url')}
      disabled={readonly}
    />
  )
}
