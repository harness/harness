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
import { Container, Text, Layout, StringSubstitute } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import cx from 'classnames'
import { defaultTo } from 'lodash-es'
import type { GitInfoProps } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { useStrings } from 'framework/strings'
import type { TypesPullReq } from 'services/code'
import { PullRequestStateLabel } from 'components/PullRequestStateLabel/PullRequestStateLabel'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { GitRefLink } from 'components/GitRefLink/GitRefLink'
import css from './PullRequestMetaLine.module.scss'

export const PullRequestMetaLine: React.FC<TypesPullReq & Pick<GitInfoProps, 'repoMetadata'>> = ({
  repoMetadata,
  target_branch,
  source_branch,
  author,
  created,
  merged,
  state,
  is_draft,
  stats
}) => {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const vars = {
    state: state,
    user: <strong>{author?.display_name || author?.email || ''}</strong>,
    commits: <strong>{stats?.commits}</strong>,
    commitsCount: stats?.commits,
    target: (
      <GitRefLink
        text={target_branch as string}
        url={routes.toCODERepository({ repoPath: repoMetadata.path as string, gitRef: target_branch })}
        showCopy
      />
    ),
    source: (
      <GitRefLink
        text={source_branch as string}
        url={routes.toCODERepository({ repoPath: repoMetadata.path as string, gitRef: source_branch })}
        showCopy
      />
    )
  }
  return (
    <Container padding={{ left: 'xlarge' }} className={css.main}>
      <Layout.Horizontal spacing="small" className={css.layout}>
        <PullRequestStateLabel data={{ is_draft, state }} />
        <Text tag="div" className={css.metaline}>
          <StringSubstitute str={getString('pr.metaLine')} vars={vars} />
        </Text>

        <PipeSeparator height={9} />
        <TimePopoverWithLocal
          time={defaultTo(merged ? merged : created, 0)}
          inline={false}
          className={cx(css.metaline, css.time)}
          font={{ variation: FontVariation.TINY }}
        />
      </Layout.Horizontal>
    </Container>
  )
}
