import React, { useMemo } from 'react'
import { Text, IconName } from '@harness/uicore'
import cx from 'classnames'
import { useStrings } from 'framework/strings'
import css from './PRCheckExecutionStatus.module.scss'

export enum PRCheckExecutionState {
  PENDING = 'pending',
  RUNNING = 'running',
  SUCCESS = 'success',
  FAILURE = 'failure',
  ERROR = 'error'
}

interface PRCheckExecutionStatusProps {
  status: PRCheckExecutionState
  iconOnly?: boolean
  noBackground?: boolean
  iconSize?: number
  className?: string
}

export const PRCheckExecutionStatus: React.FC<PRCheckExecutionStatusProps> = ({
  status,
  iconSize = 20,
  iconOnly = false,
  noBackground = false,
  className
}) => {
  const { getString } = useStrings()
  const maps = useMemo(
    () => ({
      [PRCheckExecutionState.PENDING]: {
        icon: 'ci-pending-build',
        css: css.pending,
        title: getString('pending').toLocaleUpperCase()
      },
      [PRCheckExecutionState.RUNNING]: {
        icon: 'running-filled',
        css: css.running,
        title: getString('running').toLocaleUpperCase()
      },
      [PRCheckExecutionState.SUCCESS]: {
        icon: 'execution-success',
        css: css.success,
        title: getString('success').toLocaleUpperCase()
      },
      [PRCheckExecutionState.FAILURE]: {
        icon: 'error-transparent-no-outline',
        css: css.failure,
        title: getString('failed').toLocaleUpperCase()
      },
      [PRCheckExecutionState.ERROR]: {
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
