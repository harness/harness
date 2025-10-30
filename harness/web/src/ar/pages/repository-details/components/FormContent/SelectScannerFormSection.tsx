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
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Checkbox, CheckboxVariant, Container, Layout, Text } from '@harnessio/uicore'

import { String } from '@ar/frameworks/strings'
import type { RepositoryPackageType } from '@ar/common/types'
import type { VirtualRegistryRequest } from '@ar/pages/repository-details/types'
import type { ScannerConfigSpec } from '@ar/pages/repository-details/constants'
import useCheckRequiredConfigForScan from '@ar/pages/repository-details/hooks/useCheckRequiredConfigForScan/useCheckRequiredConfigForScan'

import css from './FormContent.module.scss'

interface SelectScannerFormSectionProps {
  title: string
  packageType: RepositoryPackageType
  options: ScannerConfigSpec[]
  isEdit?: boolean
  subTitle?: string
  readonly?: boolean
}

export default function SelectScannerFormSection(props: SelectScannerFormSectionProps) {
  const { options, title, subTitle, readonly } = props
  const { setFieldValue, values } = useFormikContext<VirtualRegistryRequest>()

  const { hasRequiredLicense, hasRequiredProjectConfig, orgIdentifier, hasRequiredConfig } =
    useCheckRequiredConfigForScan()

  const handleUpdateFormikState = (event: React.FormEvent<HTMLInputElement>) => {
    const isChecked = event.currentTarget.checked
    if (isChecked) {
      setFieldValue('scanners', [...(values.scanners ?? []), { name: event.currentTarget.value }])
    } else {
      const updatedFormValue = values.scanners?.filter(each => each.name !== event.currentTarget.value)
      setFieldValue('scanners', updatedFormValue ?? [])
    }
  }

  const getCheckboxState = (value: ScannerConfigSpec['value']) => {
    return values.scanners?.some(each => each.name === value) || false
  }

  return (
    <Layout.Vertical spacing="xsmall">
      <Text font={{ variation: FontVariation.H6 }}>{title}</Text>
      {subTitle && <Text font={{ variation: FontVariation.SMALL }}>{subTitle}</Text>}
      <Container className={css.scannersContainer}>
        {options.map(each => (
          <Checkbox
            key={each.label}
            disabled={readonly || !hasRequiredConfig}
            value={each.value}
            checked={getCheckboxState(each.value)}
            variant={CheckboxVariant.BOXED}
            onChange={handleUpdateFormikState}>
            <Text
              flex={{ alignItems: 'center', justifyContent: 'flex-start' }}
              icon={each.icon}
              font={{ variation: FontVariation.FORM_LABEL }}>
              {each.label}
            </Text>
          </Checkbox>
        ))}
      </Container>
      {hasRequiredLicense && !hasRequiredProjectConfig && (
        <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} spacing="small">
          <Icon name="main-info" color={Color.PRIMARY_7} />
          <String
            useRichText
            vars={{ orgName: orgIdentifier }}
            className={css.helperText}
            stringID="repositoryDetails.repositoryForm.securityScan.containerScannerSelect.scannerNoteForRequiredConfiguration"
          />
        </Layout.Horizontal>
      )}
      {!hasRequiredLicense && (
        <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} spacing="small">
          <Icon name="main-info" color={Color.PRIMARY_7} />
          <String
            useRichText
            className={css.helperText}
            stringID="repositoryDetails.repositoryForm.securityScan.containerScannerSelect.scannerNoteForRequiredLicense"
          />
        </Layout.Horizontal>
      )}
    </Layout.Vertical>
  )
}
