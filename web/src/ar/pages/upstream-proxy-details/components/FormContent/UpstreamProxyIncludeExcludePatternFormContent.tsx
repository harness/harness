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
import { Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'

import { useStrings } from '@ar/frameworks/strings/String'
import IncludeExcludePatterns from '@ar/components/IncludeExcludePatterns/IncludeExcludePatterns'
import type { UpstreamRegistryRequest } from '../../types'

interface UpstreamProxyIncludeExcludePatternFormContentProps {
  readonly: boolean
  formikProps: FormikProps<UpstreamRegistryRequest>
  isEdit: boolean
}

export default function UpstreamProxyIncludeExcludePatternFormContent(
  props: UpstreamProxyIncludeExcludePatternFormContentProps
): JSX.Element {
  const { isEdit, readonly, formikProps } = props
  const { getString } = useStrings()
  return (
    <Layout.Vertical spacing="small">
      <Text font={{ variation: FontVariation.CARD_TITLE }}>
        {getString('repositoryDetails.repositoryForm.includeExcludePatternsTitle')}
      </Text>
      <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
        {getString('repositoryDetails.repositoryForm.includeExcludePatternsSubTitle')}
      </Text>
      <IncludeExcludePatterns<UpstreamRegistryRequest>
        isEdit={isEdit}
        disabled={readonly}
        formikProps={formikProps}
        includePatternListProps={{
          name: 'allowedPattern',
          label: getString('repositoryDetails.repositoryForm.includePatternsLabel'),
          placeholder: getString('repositoryDetails.repositoryForm.includePatternsPlaceholder'),
          addButtonLabel: getString('repositoryDetails.repositoryForm.newIncludePattern')
        }}
        excludePatternListProps={{
          name: 'blockedPattern',
          label: getString('repositoryDetails.repositoryForm.excludePatternsLabel'),
          placeholder: getString('repositoryDetails.repositoryForm.excludePatternsPlaceholder'),
          addButtonLabel: getString('repositoryDetails.repositoryForm.newExcludePattern')
        }}
      />
    </Layout.Vertical>
  )
}
