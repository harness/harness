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
import { Color } from '@harnessio/design-system'
import { Position } from '@blueprintjs/core'

import { useStrings } from '@ar/frameworks/strings/String'
import Badge from './Badge'
import css from './Badge.module.scss'

interface QuarantineBadgeProps {
  reason?: string
}

function QuarantineBadge({ reason }: QuarantineBadgeProps): JSX.Element {
  const { getString } = useStrings()
  return (
    <Badge
      className={css.quarantineBadge}
      icon="warning-icon"
      iconProps={{
        color: Color.ORANGE_900,
        size: 16
      }}
      color={Color.ORANGE_900}
      tooltip={reason}
      tooltipProps={{
        position: Position.BOTTOM,
        disabled: !reason
      }}>
      {getString('badges.quarantined')}
    </Badge>
  )
}

export default QuarantineBadge
