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
import { Layout, Text } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'

import { SecurityTestSatus } from './types'
import css from './SecurityTestsCard.module.scss'

interface SecurityTestItemProps {
  title: string
  status: SecurityTestSatus
  value: number | string
}

export default function SecurityItem(props: SecurityTestItemProps) {
  const { title, value, status } = props
  return (
    <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
      <Text font={{ variation: FontVariation.SMALL }}>{title}</Text>
      <Text
        font={{ variation: FontVariation.SMALL }}
        className={classNames(css.statusValue, {
          [css.critical]: status === SecurityTestSatus.Critical,
          [css.high]: status === SecurityTestSatus.High,
          [css.medium]: status === SecurityTestSatus.Medium,
          [css.low]: status === SecurityTestSatus.Low,
          [css.green]: status === SecurityTestSatus.Green,
          [css.info]: status === SecurityTestSatus.Info
        })}>
        {value}
      </Text>
    </Layout.Horizontal>
  )
}
