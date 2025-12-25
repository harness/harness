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
import { SimpleTagInput, TagInputProps } from '@harnessio/uicore'

import css from './MultiTagsInput.module.scss'

interface MultiTagsInputProps
  extends Omit<
    TagInputProps<
      | string
      | {
          label: string
          value: string
        }
    >,
    'labelFor' | 'keyOf' | 'itemFromNewTag'
  > {
  hidePopover?: boolean
}

export default function MultiTagsInput(props: MultiTagsInputProps) {
  const { hidePopover, className, getTagProps, ...rest } = props

  return (
    <SimpleTagInput
      className={classNames(className, css.mutliTagsInput, {
        [css.hidePopover]: props.hidePopover,
        [css.readonly]: rest.readonly
      })}
      getTagProps={(val, index, selectedItems, createdItems, items) => {
        return {
          intent: 'primary',
          ...(getTagProps ? getTagProps(val, index, selectedItems, createdItems, items) : {})
        }
      }}
      popoverProps={{ usePortal: false }}
      {...rest}
    />
  )
}
