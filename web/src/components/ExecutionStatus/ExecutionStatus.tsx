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
  ERROR = 'error'
}

interface ExecutionStatusProps {
  status: ExecutionState
  iconOnly?: boolean
  noBackground?: boolean
  iconSize?: number
  className?: string
}

export const ExecutionStatus: React.FC<ExecutionStatusProps> = ({
  status,
  iconSize = 20,
  iconOnly = false,
  noBackground = false,
  className
}) => {
  const { getString } = useStrings()
  const maps = useMemo(
    () => ({
      [ExecutionState.PENDING]: {
        icon: 'ci-pending-build',
        css: css.pending,
        title: getString('pending').toLocaleUpperCase()
      },
      [ExecutionState.RUNNING]: {
        icon: 'running-filled',
        css: css.running,
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
      }
    }),
    [getString]
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
