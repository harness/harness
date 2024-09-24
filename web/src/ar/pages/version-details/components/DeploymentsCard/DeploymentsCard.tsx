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
import { defaultTo } from 'lodash-es'
import { Link } from 'react-router-dom'
import { Card, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'

import { useAppStore } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { useParentUtils } from '@ar/hooks/useParentUtils'
import DonutChart from '@ar/components/Charts/DonutChart/DonutChart'

import css from './DeploymentsCard.module.scss'

interface DeploymentsCardProps {
  prodCount: number
  nonProdCount: number
  pipelineName?: string
  pipelineId?: string
  executionId?: string
  className?: string
  onClick?: () => void
}

export default function DeploymentsCard(props: DeploymentsCardProps) {
  const { prodCount, nonProdCount, pipelineName, executionId, onClick, className, pipelineId } = props
  const totalCount = defaultTo(prodCount, 0) + defaultTo(nonProdCount, 0)
  const { getString } = useStrings()
  const { scope } = useAppStore()
  const { getRouteToPipelineExecutionView } = useParentUtils()
  return (
    <Card className={className} onClick={onClick}>
      <Layout.Horizontal className={css.container}>
        <Layout.Vertical spacing="medium">
          <Text font={{ variation: FontVariation.CARD_TITLE }}>
            {getString('versionDetails.cards.deploymentsCard.title')}
          </Text>
          <Layout.Horizontal spacing="medium" flex={{ alignItems: 'center' }}>
            <Text font={{ variation: FontVariation.H2 }}>{totalCount}</Text>
            <DonutChart
              size={50}
              options={{
                tooltip: { enabled: false }
              }}
              items={[
                {
                  label: getString('prod'),
                  value: defaultTo(prodCount, 0),
                  color: Color.PURPLE_900
                },
                {
                  label: getString('nonProd'),
                  value: defaultTo(nonProdCount, 0),
                  color: Color.TEAL_800
                }
              ]}
            />
          </Layout.Horizontal>
        </Layout.Vertical>
        {getRouteToPipelineExecutionView && pipelineName && executionId && pipelineId && (
          <Layout.Vertical spacing="medium" flex={{ justifyContent: 'space-between', alignItems: 'flex-start' }}>
            <Text font={{ variation: FontVariation.CARD_TITLE }}>
              {getString('versionDetails.cards.deploymentsCard.buildTitle')}
            </Text>
            <Layout.Vertical spacing="xsmall">
              <Link
                to={getRouteToPipelineExecutionView({
                  accountId: scope.accountId,
                  orgIdentifier: scope.orgIdentifier,
                  projectIdentifier: scope.projectIdentifier,
                  pipelineIdentifier: pipelineId,
                  executionIdentifier: executionId,
                  module: 'ci'
                })}
                onClick={evt => {
                  evt.stopPropagation()
                }}>
                {pipelineName}
              </Link>
              <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
                {getString('versionDetails.cards.deploymentsCard.executionId')}: {executionId}
              </Text>
            </Layout.Vertical>
          </Layout.Vertical>
        )}
      </Layout.Horizontal>
    </Card>
  )
}
