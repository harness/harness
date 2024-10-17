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
import classNames from 'classnames'
import { FontVariation } from '@harnessio/design-system'
import { Card, Container, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryPackageType } from '@ar/common/types'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'

import { ContainerScannerConfig } from '../../constants'
import SelectScannerFormSection from './SelectScannerFormSection'

import css from './FormContent.module.scss'

interface SelectContainerScannersFormSectionProps {
  packageType: RepositoryPackageType
  readonly?: boolean
}

export default function SelectContainerScannersFormSection(
  props: SelectContainerScannersFormSectionProps
): JSX.Element {
  const { packageType } = props
  const { getString } = useStrings()

  const availableScannerOptions = useMemo(() => {
    const repositoryType = repositoryFactory.getRepositoryType(packageType)
    const scanners = repositoryType?.getSupportedScanners() ?? []
    return scanners.map(each => ContainerScannerConfig[each])
  }, [packageType])

  if (!availableScannerOptions.length) return <></>

  return (
    <Container>
      <Container className={css.marginTopLarge}>
        <Text className={css.cardHeading} font={{ variation: FontVariation.CARD_TITLE }}>
          {getString('repositoryDetails.repositoryForm.securityScan.title')}
        </Text>
      </Container>
      <Card className={classNames(css.cardContainer, css.marginTopLarge)}>
        <SelectScannerFormSection
          title={getString('repositoryDetails.repositoryForm.securityScan.containerScannerSelect.cardTitle')}
          subTitle={getString('repositoryDetails.repositoryForm.securityScan.containerScannerSelect.cardSubTitle')}
          packageType={packageType as RepositoryPackageType}
          options={availableScannerOptions}
          isEdit
          readonly
        />
      </Card>
    </Container>
  )
}
