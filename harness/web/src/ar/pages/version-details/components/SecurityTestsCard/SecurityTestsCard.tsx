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
import { Color, FontVariation } from '@harnessio/design-system'
import { Card, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

import SecurityItem from './SecurityItem'
import type { SecurityTestItem } from './types'

import css from './SecurityTestsCard.module.scss'

interface SecurityTestsCardProps {
  className?: string
  items: SecurityTestItem[]
  totalCount: number
  title?: string
  onClick?: () => void
}

export default function SecurityTestsCard(props: SecurityTestsCardProps) {
  const { items, title, className, totalCount, onClick } = props
  const { getString } = useStrings()

  return (
    <Card data-testid="integration-security-tests-card" className={className} onClick={onClick}>
      <Layout.Vertical spacing="medium">
        <Text font={{ variation: FontVariation.CARD_TITLE }}>
          {title ?? getString('versionDetails.cards.securityTests.title')}
        </Text>
        <Layout.Vertical className={css.container}>
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'flex-end', justifyContent: 'flex-start' }}>
            <Text font={{ variation: FontVariation.H2 }}>{totalCount}</Text>
            <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
              {getString('versionDetails.cards.securityTests.totalIssues')}
            </Text>
          </Layout.Horizontal>
          <Layout.Horizontal spacing="small">
            {items.map(each => (
              <SecurityItem key={each.value} title={each.title} status={each.status} value={each.value} />
            ))}
          </Layout.Horizontal>
        </Layout.Vertical>
      </Layout.Vertical>
    </Card>
  )
}
