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

import React, { useMemo } from 'react'
import { Text } from '@harnessio/uicore'
import type { IconName } from '@harnessio/icons'
import cx from 'classnames'
import { useStrings } from 'framework/strings'
import css from './ExecutionStatus.module.scss'

export enum ExecutionState {
  PENDING = 'pending',
  RUNNING = 'running',
  SUCCESS = 'success',
  FAILURE = 'failure',
  ERROR = 'error',
  SKIPPED = 'skipped',
  KILLED = 'killed'
}

interface ExecutionStatusProps {
  status: ExecutionState
  iconOnly?: boolean
  noBackground?: boolean
  iconSize?: number
  className?: string
  isCi?: boolean
  inExecution?: boolean
  inPr?: boolean
}

export enum ExecutionStateExtended {
  FAILED = 'failed',
  ABORTED = 'aborted',
  ASYNCWAITING = 'asyncwaiting'
}

export const ExecutionStatus: React.FC<ExecutionStatusProps> = ({
  status,
  iconSize = 20,
  iconOnly = false,
  noBackground = false,
  className,
  isCi = false,
  inExecution = false,
  inPr = false
}) => {
  const { getString } = useStrings()
  const maps = useMemo(
    () => ({
      [ExecutionState.PENDING]: {
        icon: isCi ? (inExecution ? 'execution-waiting' : 'running-filled') : 'ci-pending-build',
        css: isCi ? (inExecution ? css.waiting : inPr ? css.prWaiting : css.executionWaiting) : css.pending,
        title: getString('pending').toLocaleUpperCase()
      },
      [ExecutionState.RUNNING]: {
        icon: 'running-filled',
        css: isCi ? css.runningBlue : css.running,
        title: getString('running').toLocaleUpperCase()
      },
      [ExecutionState.SUCCESS]: {
        icon: 'execution-success',
        css: css.success,
        title: getString('success').toLocaleUpperCase()
      },
      [ExecutionStateExtended.FAILED]: {
        icon: 'error-transparent-no-outline',
        css: css.failure,
        title: getString('failed').toLocaleUpperCase()
      },
      [ExecutionState.FAILURE]: {
        icon: inPr ? 'error-transparent-no-outline' : 'warning-icon',
        css: css.failure,
        title: getString('failed').toLocaleUpperCase()
      },
      [ExecutionState.ERROR]: {
        icon: 'solid-error',
        css: css.error,
        title: getString('error').toLocaleUpperCase()
      },
      [ExecutionState.SKIPPED]: {
        icon: 'execution-timeout',
        css: null,
        title: getString('skipped').toLocaleUpperCase()
      },
      [ExecutionState.KILLED]: {
        icon: 'execution-stopped',
        css: null,
        title: getString('killed').toLocaleUpperCase()
      },
      [ExecutionStateExtended.ABORTED]: {
        icon: 'execution-stopped',
        css: null,
        title: getString('killed').toLocaleUpperCase()
      },
      [ExecutionStateExtended.ASYNCWAITING]: {
        icon: 'running-filled',
        css: css.running,
        title: getString('running').toLocaleUpperCase()
      }
    }),
    [getString, inExecution, isCi]
  )
  const map = useMemo(() => {
    if (!maps || !status || !maps[status])
      return {
        icon: '',
        css: null,
        title: ''
      }
    return maps[status]
  }, [maps, status])
  return (
    <Text
      tag="span"
      className={cx(css.main, map?.css, { [css.iconOnly]: iconOnly, [css.noBackground]: noBackground }, className)}
      icon={map?.icon as IconName}
      iconProps={{ size: iconOnly ? iconSize : 12 }}>
      {!iconOnly && map?.title}
    </Text>
  )
}
