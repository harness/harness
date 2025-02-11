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
import type { IconName } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Layout, Text, TextProps } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings/String'
import CommandBlock from '@ar/components/CommandBlock/CommandBlock'
import CopyButton from '@ar/components/CopyButton/CopyButton'

import { LabelValueTypeEnum } from './type'

interface CopyTextProps extends TextProps {
  value: string
}

export function CopyText(props: CopyTextProps): JSX.Element {
  return (
    <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} spacing="small">
      <Text {...props}>{props.value}</Text>
      <CopyButton textToCopy={props.value} iconProps={{ color: Color.PRIMARY_7 }} />
    </Layout.Horizontal>
  )
}

interface LabelValueProps {
  label: string
  value: string | number | undefined
  type: LabelValueTypeEnum
  icon?: IconName
}

export function LabelValueContent(props: LabelValueProps): JSX.Element {
  const { label, value, type, icon } = props
  const { getString } = useStrings()
  const transformedValue = defaultTo(value, getString('na'))
  const renderValue = () => {
    switch (type) {
      case LabelValueTypeEnum.CommandBlock:
        return <CommandBlock noWrap commandSnippet={transformedValue.toString()} allowCopy />
      case LabelValueTypeEnum.CopyText:
        return (
          <CopyText
            lineClamp={1}
            value={transformedValue.toString()}
            font={{ variation: FontVariation.BODY2, weight: 'light' }}
          />
        )
      case LabelValueTypeEnum.PackageType:
        return (
          <Text icon={icon} iconProps={{ size: 20 }} font={{ variation: FontVariation.BODY2, weight: 'light' }}>
            {transformedValue}
          </Text>
        )

      default:
        return (
          <Text lineClamp={1} font={{ variation: FontVariation.BODY2, weight: 'light' }}>
            {transformedValue}
          </Text>
        )
    }
  }
  return (
    <>
      <Text font={{ variation: FontVariation.BODY2, weight: 'semi-bold' }}>{label}</Text>
      {renderValue()}
    </>
  )
}
