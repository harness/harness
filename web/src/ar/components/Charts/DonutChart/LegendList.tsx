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
import { Layout, Utils, Text } from '@harnessio/uicore'
import { FontVariation, type Color } from '@harnessio/design-system'

import type { PieChartItem } from '../types'
import css from './DonutChart.module.scss'

export const Legend = ({ label, color }: { label: React.ReactNode; color?: Color }) => {
  return (
    <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }}>
      <div className={css.legendColor} style={{ background: Utils.getRealCSSColor(color || '') }}></div>
      {label}
    </Layout.Horizontal>
  )
}

export const LegendList = ({ items }: { items: PieChartItem[] }) => {
  return (
    <div>
      <Layout.Vertical>
        {items.map(item => {
          return (
            <Layout.Horizontal spacing={'xsmall'} key={item.value}>
              <Legend label={<Text font={{ variation: FontVariation.SMALL }}>{item.label}</Text>} color={item.color} />
              <Text font={{ variation: FontVariation.SMALL }}>({item.value})</Text>
            </Layout.Horizontal>
          )
        })}
      </Layout.Vertical>
    </div>
  )
}
