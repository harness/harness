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
import { Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'

import { ThumbnailTagEnum, ThumbnailTags } from '../Tag/ThumbnailTags'

import css from './ThumbnailLabel.module.scss'

interface ThumbnailLabelProps {
  label: string
  disabled?: boolean
  tag?: ThumbnailTagEnum
}

export default function ThumbnailLabel(props: ThumbnailLabelProps): JSX.Element {
  const { label, tag, disabled } = props
  return (
    <div>
      <Text
        font={{ size: 'small', weight: 'semi-bold' }}
        color={disabled ? Color.GREY_500 : Color.GREY_600}
        margin={{ top: 'small' }}>
        {label}
      </Text>
      {tag && <ThumbnailTags className={css.tag} tag={tag} />}
    </div>
  )
}
