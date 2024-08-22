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
import { FontVariation } from '@harnessio/design-system'
import { CopyToClipboard, Layout, Text, TextProps } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings/String'
import CommandBlock from '@ar/components/CommandBlock/CommandBlock'

interface CopyTextProps extends TextProps {
  value: string
}

export function CopyText(props: CopyTextProps): JSX.Element {
  return (
    <Layout.Horizontal spacing="small">
      <Text {...props}>{props.value}</Text>
      <CopyToClipboard content={props.value} showFeedback />
    </Layout.Horizontal>
  )
}

interface LabelValueProps {
  label: string
  value: string | number | undefined
  withCopyText?: boolean
  withCodeBlock?: boolean
}

export function LabelValueContent(props: LabelValueProps): JSX.Element {
  const { label, value, withCopyText, withCodeBlock } = props
  const { getString } = useStrings()
  const transformedValue = defaultTo(value, getString('na'))
  const renderValue = () => {
    if (withCopyText)
      return <CopyText lineClamp={1} value={transformedValue.toString()} font={{ variation: FontVariation.SMALL }} />
    if (withCodeBlock) return <CommandBlock noWrap commandSnippet={transformedValue.toString()} allowCopy />
    return (
      <Text lineClamp={1} font={{ variation: FontVariation.SMALL }}>
        {transformedValue}
      </Text>
    )
  }
  return (
    <>
      <Text font={{ variation: FontVariation.SMALL_BOLD }}>{label}</Text>
      {renderValue()}
    </>
  )
}
