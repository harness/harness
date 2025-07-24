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

import { ColorName } from 'utils/Utils'
import { Label } from 'components/Label/Label'
import type { TypesLabelValue } from 'services/code'

import { useStrings } from '@ar/frameworks/strings'

interface LabelValuesListContentProps {
  labelName?: string
  scope?: number
  color?: ColorName
  list: TypesLabelValue[]
  allowDynamicValues?: boolean
  defaultLabelName?: string
}

export function LabelValuesListContent(props: LabelValuesListContentProps) {
  const { list, allowDynamicValues, defaultLabelName, labelName, scope, color } = props
  const { getString } = useStrings()
  if (!list.length && !allowDynamicValues) {
    return (
      <Label
        name={labelName || defaultLabelName || getString('labelsList.defaultLabelName')}
        label_color={color || ColorName.Blue}
      />
    )
  }
  return (
    <>
      {list.map(value => (
        <Label
          key={`${labelName}-${value.value}`}
          name={labelName || ''}
          scope={scope}
          label_value={{ name: value.value, color: value.color as ColorName }}
        />
      ))}
      {allowDynamicValues && (
        <Label
          name={labelName || defaultLabelName || getString('labelsList.defaultLabelName')}
          label_color={color || ColorName.Blue}
          label_value={{ name: getString('labelsList.canbeAddedByUsers') }}
        />
      )}
    </>
  )
}

export default LabelValuesListContent
