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
import { defaultTo } from 'lodash-es'
import { Color } from '@harnessio/design-system'
import { Icon, IconProps } from '@harnessio/icons'

import css from './MultiTagsInput.module.scss'

export default function TagIcon(props: Partial<IconProps>) {
  return (
    <Icon
      {...props}
      color={defaultTo(props.color, Color.PRIMARY_7)}
      name={defaultTo(props.name, 'tag')}
      className={classNames(css.tagIcon, props.className)}
    />
  )
}
