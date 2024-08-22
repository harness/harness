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
import { Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'

import { useStrings } from '@ar/frameworks/strings/String'
import type { VirtualRegistryRequest } from '@ar/pages/repository-details/types'
import type { UpstreamProxyPackageType } from '@ar/pages/upstream-proxy-details/types'

import UpstreamProxiesSelect from '../UpstreamProxiesSelect'
import css from './FormContent.module.scss'

interface RepositoryUpstreamProxiesFormContentProps {
  isEdit: boolean
  disabled: boolean
}

function RepositoryUpstreamProxiesFormContent(
  props: RepositoryUpstreamProxiesFormContentProps & { formik: FormikContextType<VirtualRegistryRequest> }
): JSX.Element {
  const { formik, isEdit, disabled } = props
  const { getString } = useStrings()
  const { values } = formik
  const { packageType } = values

  return (
    <Layout.Vertical flex={{ alignItems: 'flex-start' }} spacing="xsmall">
      <Text font={{ variation: FontVariation.CARD_TITLE }}>
        {getString('repositoryDetails.repositoryForm.upstreamProxiesTitle')}
      </Text>
      <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
        {getString('repositoryDetails.repositoryForm.upstreamProxiesSubTitle')}
      </Text>
      <UpstreamProxiesSelect
        className={css.upstreamProxiesWrapper}
        name="config.upstreamProxies"
        formikProps={formik}
        isEdit={isEdit}
        disabled={disabled}
        packageType={packageType as UpstreamProxyPackageType}
      />
    </Layout.Vertical>
  )
}

export default connect<RepositoryUpstreamProxiesFormContentProps, VirtualRegistryRequest>(
  RepositoryUpstreamProxiesFormContent
)
