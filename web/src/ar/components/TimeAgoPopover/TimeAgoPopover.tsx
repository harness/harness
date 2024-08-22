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
import ReactTimeago from 'react-timeago'
import type { IconName } from '@harnessio/icons'
import { Text, Popover, TextProps } from '@harnessio/uicore'
import { Classes, IPopoverProps, PopoverInteractionKind, Position } from '@blueprintjs/core'

import { DateTimeContent } from './DateTimeContent'

interface TimeAgoPopoverProps extends TextProps {
  time: number
  popoverProps?: IPopoverProps
  icon?: IconName
  className?: string
}

export const TimeAgoPopover: React.FC<TimeAgoPopoverProps> = props => {
  const { time, popoverProps, icon, className, ...textProps } = props
  return (
    <Popover
      interactionKind={PopoverInteractionKind.HOVER}
      position={Position.TOP}
      className={Classes.DARK}
      modifiers={{ preventOverflow: { escapeWithReference: true } }}
      {...popoverProps}>
      <Text inline {...textProps} icon={icon} className={className}>
        <ReactTimeago date={time} live title={''} />
      </Text>
      <DateTimeContent time={time} padding={'medium'} />
    </Popover>
  )
}
