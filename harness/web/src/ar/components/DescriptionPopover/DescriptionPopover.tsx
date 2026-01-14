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
import { IPopoverProps, PopoverInteractionKind } from '@blueprintjs/core'
import { Icon, type IconProps } from '@harnessio/icons'
import { Text, Popover, Container } from '@harnessio/uicore'
import { useStrings } from '@ar/frameworks/strings/String'
import css from './DescriptionPopover.module.scss'

interface DescriptionPopoverProps {
  text: string
  className?: string
  popoverProps?: IPopoverProps
  iconProps?: Omit<IconProps, 'name'>
  target?: React.ReactElement
}

const DescriptionPopover: React.FC<DescriptionPopoverProps> = (props): JSX.Element => {
  const { text, popoverProps, iconProps, target } = props

  const { getString } = useStrings()
  return (
    <Popover interactionKind={PopoverInteractionKind.HOVER} {...popoverProps}>
      {target || <Icon name="description" {...iconProps} size={iconProps?.size || 18} />}
      <Container padding="medium">
        <Text font={{ size: 'normal', weight: 'bold' }}>{getString('description')}</Text>
        <Container className={css.descriptionPopover}>{text}</Container>
      </Container>
    </Popover>
  )
}

export default DescriptionPopover
