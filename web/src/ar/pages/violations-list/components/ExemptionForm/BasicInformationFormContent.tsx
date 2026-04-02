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

import React, { useMemo } from 'react'
import { useFormikContext } from 'formik'
import { FontVariation } from '@harnessio/design-system'
import { useListFirewallExceptionVersionsV3Query } from '@harnessio/react-har-service-client'
import { Container, FormInput, Layout, MultiSelectOption, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { useAppStore } from '@ar/hooks'
import type { ExemptionFormSpec } from './types'

interface BasicInformationFormContentProps {
  registryId: string
  packageName: string
  isEdit?: boolean
}

export default function BasicInformationFormContent(props: BasicInformationFormContentProps) {
  const { getString } = useStrings()
  const { scope } = useAppStore()
  const { setFieldValue } = useFormikContext<ExemptionFormSpec>()

  const { data, isFetching, error } = useListFirewallExceptionVersionsV3Query({
    queryParams: {
      account_identifier: scope.accountId || '',
      registry_id: props.registryId,
      package_name: props.packageName,
      page: 0,
      size: 100
    }
  })

  const versionOptions: MultiSelectOption[] = useMemo(() => {
    if (isFetching) {
      return [
        {
          label: getString('loading'),
          value: ''
        }
      ]
    }
    if (error) {
      return [
        {
          label: error.error?.message || getString('failedToLoadData'),
          value: ''
        }
      ]
    }
    if (!data?.content?.items) {
      return []
    }
    return data.content.items.map(item => ({
      label: item,
      value: item
    }))
  }, [data, isFetching, error, getString])

  return (
    <Layout.Vertical spacing="medium">
      <Text font={{ variation: FontVariation.CARD_TITLE }}>
        {getString('violationsList.createExemptionForm.basicInformationSection.title')}
      </Text>
      <Container>
        <FormInput.Text
          disabled
          name="packageName"
          label={getString('violationsList.createExemptionForm.basicInformationSection.packageName')}
          placeholder={getString('violationsList.createExemptionForm.basicInformationSection.packageName')}
        />
        <FormInput.MultiSelect
          items={versionOptions}
          disabled={props.isEdit}
          usePortal
          name="versionList"
          label={getString('violationsList.createExemptionForm.basicInformationSection.version')}
          placeholder={getString('violationsList.createExemptionForm.basicInformationSection.version')}
          onChange={opts => {
            setFieldValue(
              'versionList',
              opts.filter(each => each.value !== '')
            )
          }}
          multiSelectProps={{
            allowCreatingNewItems: true,
            allowCommaSeparatedList: true,
            itemDisabled: () => isFetching || !!error
          }}
        />
      </Container>
    </Layout.Vertical>
  )
}
