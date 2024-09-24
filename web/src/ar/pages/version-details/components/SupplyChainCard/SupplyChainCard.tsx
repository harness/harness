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
import classNames from 'classnames'
import { Card, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'

import { useStrings } from '@ar/frameworks/strings'

import SecurityItem from '../SecurityTestsCard/SecurityItem'
import { SecurityTestSatus } from '../SecurityTestsCard/types'

import css from './SupplyChainCard.module.scss'

interface SupplyChainCardProps {
  className?: string
  title?: string
  totalComponents: number
  allowListCount: number
  denyListCount: number
  sbomScore: string | number
  onClick?: () => void
}

export default function SupplyChainCard(props: SupplyChainCardProps) {
  const { title, totalComponents, allowListCount, denyListCount, className, sbomScore, onClick } = props
  const { getString } = useStrings()

  return (
    <Card className={className} onClick={onClick}>
      <Layout.Vertical>
        <Text font={{ variation: FontVariation.CARD_TITLE }}>
          {title ?? getString('versionDetails.cards.supplyChain.title')}
        </Text>
        <Layout.Horizontal className={css.container}>
          <Layout.Vertical className={css.column}>
            <Text font={{ variation: FontVariation.H2 }}>{totalComponents}</Text>
            <Text font={{ variation: FontVariation.SMALL }}>
              {getString('versionDetails.cards.supplyChain.totalComponents')}
            </Text>
          </Layout.Vertical>
          <Layout.Vertical className={css.column} spacing="small">
            <SecurityItem
              title={getString('versionDetails.cards.supplyChain.sbomScore')}
              value={sbomScore}
              status={SecurityTestSatus.Green}
            />
            <Text
              flex={{ alignItems: 'center' }}
              font={{ variation: FontVariation.SMALL }}
              rightIcon="download-manifests"
              iconProps={{ size: 18 }}>
              {getString('versionDetails.cards.supplyChain.slsaProvenance')}
            </Text>
          </Layout.Vertical>
        </Layout.Horizontal>
        <Layout.Horizontal className={css.container}>
          <Text
            className={css.column}
            color={Color.PRIMARY_7}
            font={{ variation: FontVariation.SMALL }}
            icon="danger-icon"
            iconProps={{ size: 18 }}>
            {allowListCount} {getString('versionDetails.cards.supplyChain.allowList')}
          </Text>
          <Text
            className={classNames(css.column, css.primaryColumn)}
            color={Color.PRIMARY_7}
            font={{ variation: FontVariation.SMALL }}>
            {denyListCount} {getString('versionDetails.cards.supplyChain.denyListViolation')}
          </Text>
        </Layout.Horizontal>
      </Layout.Vertical>
    </Card>
  )
}
