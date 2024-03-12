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
      icon: CodeIcon.Rejected,
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
