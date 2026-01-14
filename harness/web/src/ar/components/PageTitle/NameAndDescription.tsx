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
import { Color } from '@harnessio/design-system'
import { Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import HeaderTitle from '@ar/components/Header/Title'

interface NameAndDescriptionProps {
  name: string
  description?: string
  hideDescription?: boolean
}

export default function NameAndDescription(props: NameAndDescriptionProps): JSX.Element {
  const { getString } = useStrings()
  const { name, description, hideDescription = false } = props
  return (
    <Layout.Vertical spacing="small">
      <HeaderTitle>{name}</HeaderTitle>
      {!hideDescription && (
        <Text font={{ size: 'small' }} color={Color.GREY_500} width={800} lineClamp={1}>
          {description || getString('noDescription')}
        </Text>
      )}
    </Layout.Vertical>
  )
}
