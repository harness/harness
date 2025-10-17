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
import { ITagProps, PopoverInteractionKind } from '@blueprintjs/core'
import { Container, Layout, Popover, Tag, Text } from '@harnessio/uicore'
import type { ListTagsProps } from '@harnessio/uicore/dist/components/TagsPopover/TagsPopover'

import TagIcon from '@ar/components/MultiTagsInput/TagIcon'

import css from './LabelsPopover.module.scss'

interface LabelsPopoverProps extends Omit<ListTagsProps, 'tags'> {
  labels: string[]
  tagProps?: ITagProps
  withCount?: boolean
}

function LabelsPopover(props: LabelsPopoverProps): JSX.Element {
  const {
    labels,
    className,
    iconProps,
    popoverProps,
    containerClassName,
    tagsTitle,
    tagClassName,
    tagProps,
    withCount = false
  } = props
  if (!labels.length) return <></>
  return (
    <Popover interactionKind={PopoverInteractionKind.HOVER} className={css.popover} {...popoverProps}>
      <Layout.Horizontal className={className} flex={{ align: 'center-center' }} spacing="xsmall">
        <TagIcon {...iconProps} size={iconProps?.size || 15} />
        {withCount && <Text>{labels.length}</Text>}
      </Layout.Horizontal>
      <Container padding="small" className={containerClassName}>
        <Text font={{ size: 'small', weight: 'bold' }}>{tagsTitle}</Text>
        <Container className={css.labelsPopover}>
          {labels.map(label => {
            return (
              <Tag
                className={classNames(css.label, tagClassName, {
                  [css.interactive]: !!tagProps?.interactive
                })}
                key={label}
                aria-valuetext={label}
                {...tagProps}>
                {label}
              </Tag>
            )
          })}
        </Container>
      </Container>
    </Popover>
  )
}

export default LabelsPopover
