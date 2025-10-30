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

import React, { PropsWithChildren } from 'react'
import classNames from 'classnames'
import { Tag } from '@blueprintjs/core'
import { Utils } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

import css from './Tag.module.scss'

export function ProdTag() {
  const { getString } = useStrings()
  return (
    <Tag className={classNames(css.tag, css.prodTag)} round>
      {getString('prod')}
    </Tag>
  )
}

export function NonProdTag() {
  const { getString } = useStrings()
  return (
    <Tag className={classNames(css.tag, css.nonProdTag)} round>
      {getString('nonProd')}
    </Tag>
  )
}

interface DonutChartTagProps {
  color?: string
  backgroundColor?: string
}

export function DonutChartTag(props: PropsWithChildren<DonutChartTagProps>) {
  return (
    <Tag
      className={classNames(css.tag)}
      style={{
        backgroundColor: Utils.getRealCSSColor(props.backgroundColor as string),
        color: Utils.getRealCSSColor(props.color as string)
      }}
      round>
      {props.children}
    </Tag>
  )
}
