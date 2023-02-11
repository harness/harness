import React from 'react'
import { Text, StringSubstitute, IconName } from '@harness/uicore'
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
      icon: 'success-tick',
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
