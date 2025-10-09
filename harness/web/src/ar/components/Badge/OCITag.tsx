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
import { Text } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'

import Badge from './Badge'
import css from './Badge.module.scss'

interface OCITag {
  tag: string
  onClick?: React.MouseEventHandler<HTMLDivElement>
}

export default function OCITag({ tag, onClick }: OCITag) {
  return (
    <Badge
      className={classNames(css.dockerTag, {
        [css.hover]: !!onClick
      })}
      onClick={onClick}>
      <Text lineClamp={1} font={{ variation: FontVariation.SMALL }}>
        {tag}
      </Text>
    </Badge>
  )
}
