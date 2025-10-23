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

import { StringKeys, useStrings } from '@ar/frameworks/strings'

import Tag from './Tag'

import css from './Tag.module.scss'

export enum ThumbnailTagEnum {
  Beta = 'beta',
  New = 'new',
  ComingSoon = 'comingSoon'
}

const TagToConfigMap: Record<ThumbnailTagEnum, { label: StringKeys; className: keyof typeof css }> = {
  beta: { label: 'beta', className: 'betaTag' },
  new: { label: 'new', className: 'newTag' },
  comingSoon: { label: 'soon', className: 'comingSoonTag' }
}

interface ThumbnailTagsProps {
  tag: ThumbnailTagEnum
  className?: string
}

export function ThumbnailTags(props: ThumbnailTagsProps): JSX.Element {
  const { tag, className } = props
  const { getString } = useStrings()
  const config = TagToConfigMap[tag]
  return (
    <Tag className={classNames(className, css.tag, css[config.className])} round>
      {getString(config.label)}
    </Tag>
  )
}
