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
}

export const ExecutionStatus: React.FC<ExecutionStatusProps> = ({
  status,
  iconSize = 20,
  iconOnly = false,
  noBackground = false,
  className,
  isCi = false,
  inExecution = false
}) => {
  const { getString } = useStrings()
  const maps = useMemo(
    () => ({
      [ExecutionState.PENDING]: {
        icon: isCi ? (inExecution ? 'execution-waiting' : 'running-filled') : 'ci-pending-build',
        css: isCi ? (inExecution ? css.waiting : css.executionWaiting) : css.pending,
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
      [ExecutionState.FAILURE]: {
        icon: 'error-transparent-no-outline',
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
      }
    }),
    [getString, isCi]
  )
  const map = useMemo(() => maps[status], [maps, status])

  return (
    <Text
      tag="span"
      className={cx(css.main, map.css, { [css.iconOnly]: iconOnly, [css.noBackground]: noBackground }, className)}
      icon={map.icon as IconName}
      iconProps={{ size: iconOnly ? iconSize : 12 }}>
      {!iconOnly && map.title}
    </Text>
  )
}
