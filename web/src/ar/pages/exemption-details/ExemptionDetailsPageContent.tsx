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

import React, { useContext } from 'react'
import classNames from 'classnames'
import { FontVariation } from '@harnessio/design-system'
import { Card, Container, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { Separator } from '@ar/components/Separator/Separator'

import { ExemptionDetailsContext } from './context/ExemptionDetailsProvider'

import DependencyDetailsSection from './components/ExemptionDetailsSection/DependencyDetailsSection'
import ExemptionDetailsSection from './components/ExemptionDetailsSection/ExemptionDetailsSection'
import css from './ExemptionDetailsPage.module.scss'

function ExemptionDetailsPageContent() {
  const { getString } = useStrings()
  const { data } = useContext(ExemptionDetailsContext)
  return (
    <Container padding="large">
      <Card className={classNames(css.cardContainer)}>
        <Text className={css.cardHeading} font={{ variation: FontVariation.CARD_TITLE }}>
          {getString('exemptionDetails.cards.section1.title')}
        </Text>
        <DependencyDetailsSection data={data} />
        <Separator />
        <Text className={css.cardHeading} font={{ variation: FontVariation.CARD_TITLE }}>
          {getString('exemptionDetails.cards.section2.title')}
        </Text>
        <ExemptionDetailsSection data={data} />
      </Card>
    </Container>
  )
}

export default ExemptionDetailsPageContent
