import React from 'react'
import { Text, StringSubstitute, IconName } from '@harness/uicore'
import cx from 'classnames'
import { CodeIcon } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import type { TypesPullReq } from 'services/code'
import css from './PullRequestStateLabel.module.scss'

export const PullRequestStateLabel: React.FC<{ data: TypesPullReq; iconSize?: number; iconOnly?: boolean }> = ({
  data,
  iconSize = 20,
  iconOnly = false
}) => {
  const { getString } = useStrings()
  const maps = {
    open: {
      icon: CodeIcon.PullRequest,
      css: css.open
    },
    merged: {
      icon: CodeIcon.Merged,
      css: css.merged
    },
    closed: {
      icon: CodeIcon.Merged,
      css: css.closed
    },
    draft: {
      icon: CodeIcon.Draft,
      css: css.draft
    },
    unknown: {
      icon: CodeIcon.PullRequest,
      css: css.open
    }
  }
  const map = data.is_draft ? maps.draft : maps[data.state || 'unknown']

  return (
    <Text
      tag="span"
      className={cx(css.prStatus, map.css, { [css.iconOnly]: iconOnly })}
      icon={map.icon as IconName}
      iconProps={{ size: iconOnly ? iconSize : 12 }}>
      {!iconOnly && (
        <StringSubstitute str={getString('pr.state')} vars={{ state: data.is_draft ? 'draft' : data.state }} />
      )}
    </Text>
  )
}
