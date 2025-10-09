/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import { Text, StringSubstitute } from '@harnessio/uicore'
import type { IconName } from '@harnessio/icons'
import cx from 'classnames'
import { CodeIcon } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import css from './ExecutionStatusLabel.module.scss'

export type EnumPullReqExecutionState = 'success' | 'failed' | 'unknown'
export const ExecutionStatusLabel: React.FC<{
  data: { state?: EnumPullReqExecutionState }
  iconSize?: number
  iconOnly?: boolean
}> = ({ data, iconSize = 20, iconOnly = false }) => {
  const { getString } = useStrings()
  const maps = {
    unknown: {
      icon: CodeIcon.PullRequest,
      css: css.open
    },
    success: {
      icon: 'execution-success',
      css: css.success
    },
    failed: {
      icon: 'danger-icon',
      css: css.failure
    }
  }
  const map = maps[data.state || 'unknown']

  return (
    <Text
      tag="span"
      className={cx(css.executionStatus, map.css, { [css.iconOnly]: iconOnly })}
      icon={map.icon as IconName}
      iconProps={{ size: iconOnly ? iconSize : 14 }}>
      {!iconOnly && <StringSubstitute str={getString('pr.executionState')} vars={{ state: data.state }} />}
    </Text>
  )
}
