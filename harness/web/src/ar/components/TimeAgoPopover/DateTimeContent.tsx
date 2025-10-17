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
import { upperCase } from 'lodash-es'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Color, StyledProps } from '@harnessio/design-system'

import { useStrings } from '@ar/frameworks/strings/String'

type DateTimeContentProps = {
  time: number
  padding?: StyledProps['padding']
}

export const DateTimeContent = ({ time, padding }: DateTimeContentProps): JSX.Element => {
  const { getString } = useStrings()

  return (
    <Container padding={padding}>
      <Layout.Horizontal flex={{ justifyContent: 'flex-start' }} spacing={'small'}>
        <Text font={{ size: 'xsmall', weight: 'semi-bold' }} color={Color.GREY_200}>
          {upperCase(getString('dateLabel'))}
        </Text>
        <Text font={{ size: 'small', weight: 'bold' }} color={Color.WHITE}>
          {new Date(time).toLocaleDateString()}
        </Text>
      </Layout.Horizontal>
      <Layout.Horizontal margin={{ top: 'small' }} flex={{ justifyContent: 'flex-start' }} spacing={'small'}>
        <Text font={{ size: 'xsmall', weight: 'semi-bold' }} color={Color.GREY_200}>
          {upperCase(getString('timeLabel'))}
        </Text>
        <Text font={{ size: 'small', weight: 'bold' }} color={Color.WHITE}>
          {new Date(time).toLocaleTimeString()}
        </Text>
      </Layout.Horizontal>
    </Container>
  )
}
