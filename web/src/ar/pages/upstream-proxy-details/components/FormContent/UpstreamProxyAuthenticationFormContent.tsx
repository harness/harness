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
import { Layout } from '@harnessio/uicore'

import RepositoryUrlInput from '../RepositoryUrlInput/RepositoryUrlInput'
import AuthenticationFormInput from '../AuthenticationFormInput/AuthenticationFormInput'

interface UpstreamProxyAuthenticationFormContentProps {
  readonly: boolean
}

export default function UpstreamProxyAuthenticationFormContent({
  readonly
}: UpstreamProxyAuthenticationFormContentProps): JSX.Element {
  return (
    <Layout.Vertical data-testid="upstream-source-auth-definition" spacing="small">
      <RepositoryUrlInput readonly={readonly} />
      <AuthenticationFormInput readonly={readonly} />
    </Layout.Vertical>
  )
}
