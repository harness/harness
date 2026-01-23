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
import type { IconName } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Card, Layout, Text, TextProps } from '@harnessio/uicore'
import css from './TableCard.module.scss'

interface TableCardProps {
  title: string
  titleIcon?: IconName
  iconProps?: TextProps['iconProps']
  value: string
  subText?: string
  onClick?: () => void
  active?: boolean
}

function TableCard({ title, titleIcon, iconProps, value, onClick, subText, active }: TableCardProps) {
  return (
    <Card
      onClick={onClick}
      className={classNames(css.tableCard, {
        [css.clickable]: !!onClick,
        [css.active]: !!active
      })}>
      <Layout.Vertical>
        <Text icon={titleIcon} iconProps={iconProps} font={{ variation: FontVariation.BODY }} color={Color.GREY_700}>
          {title}
        </Text>
        <Text font={{ variation: FontVariation.H3, weight: 'bold' }} color={Color.GREY_700}>
          {value}
        </Text>
        {subText && (
          <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_400}>
            {subText}
          </Text>
        )}
      </Layout.Vertical>
    </Card>
  )
}

export default TableCard
