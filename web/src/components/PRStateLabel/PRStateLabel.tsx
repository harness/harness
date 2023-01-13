import React from 'react'
import { Text, Color, StringSubstitute, IconName } from '@harness/uicore'
import cx from 'classnames'
import { CodeIcon, PullRequestState } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import css from './PRStateLabel.module.scss'

export const PRStateLabel: React.FC<{ state: PullRequestState }> = ({ state }) => {
  const { getString } = useStrings()

  let color = Color.GREEN_700
  let icon: IconName = CodeIcon.PullRequest
  let clazz: typeof css | string = ''

  switch (state) {
    case PullRequestState.MERGED:
      color = Color.PURPLE_700
      icon = CodeIcon.PullRequest
      clazz = css.merged
      break
    case PullRequestState.CLOSED:
      color = Color.GREY_600
      icon = CodeIcon.PullRequest
      clazz = css.closed
      break
    case PullRequestState.REJECTED:
      color = Color.RED_600
      icon = CodeIcon.PullRequestRejected
      clazz = css.rejected
      break
    default:
      break
  }

  return (
    <Text inline className={cx(css.state, clazz)} icon={icon} iconProps={{ color, size: 9 }}>
      <StringSubstitute str={getString('pr.state')} vars={{ state }} />
    </Text>
  )
}
