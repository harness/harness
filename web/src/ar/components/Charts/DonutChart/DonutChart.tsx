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
import Highcharts from 'highcharts'
import HighchartsReact from 'highcharts-react-official'
import { Layout, Utils } from '@harnessio/uicore'

import { LegendList } from './LegendList'
import { getParsedOptions } from '../utils'
import type { PieChartItem } from '../types'

interface DonutChartProps {
  items: PieChartItem[]
  size?: number
  height?: number
  width?: number
  options?: Highcharts.Options
  backgroundColor?: string
}

export default function DonutChart(props: DonutChartProps) {
  const { size, items, backgroundColor, options = {} } = props
  const defaultOptions: Highcharts.Options = useMemo(
    () => ({
      chart: {
        type: 'pie',
        width: size,
        height: size,
        backgroundColor,
        margin: [0, 0, 0, 0],
        spacing: [0, 0, 0, 0]
      },
      title: {
        text: ''
      },
      credits: { enabled: false },
      tooltip: {
        useHTML: true,
        padding: 4,
        outside: true,
        formatter: function () {
          const { point } = this as { point: { name: string; y: number } }
          return `${point.name}: ${point.y}`
        }
      },
      plotOptions: {
        pie: {
          size,
          dataLabels: {
            enabled: false
          },
          innerSize: '60%',
          depth: 15,
          states: {
            hover: {
              halo: null
            }
          }
        }
      },
      series: [
        {
          animation: false,
          type: 'pie',
          colorByPoint: true,
          data: items.map(item => ({
            name: item.label,
            y: item.value,
            color: Utils.getRealCSSColor(item.color as string)
          })),
          cursor: 'pointer'
        }
      ]
    }),
    [size, items]
  )

  const parsedOptions = useMemo(() => getParsedOptions(defaultOptions, options), [defaultOptions, options])
  return (
    <Layout.Horizontal spacing="medium" flex={{ alignItems: 'center' }}>
      <HighchartsReact highcharts={Highcharts} options={parsedOptions}></HighchartsReact>
      <LegendList items={items} />
    </Layout.Horizontal>
  )
}
