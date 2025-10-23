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
import type { TypesListCommitResponse } from 'services/code'
import type { GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { PullRequestSection } from 'utils/Utils'
import { PullRequestTabContentWrapper } from '../PullRequestTabContentWrapper'

interface PullRequestCommitsProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullReqMetadata'> {
  pullReqCommits?: TypesListCommitResponse
}

export const PullRequestCommits: React.FC<PullRequestCommitsProps> = ({
  repoMetadata,
  pullReqMetadata,
  pullReqCommits
}) => {
  const { getString } = useStrings()

  return (
    <PullRequestTabContentWrapper section={PullRequestSection.COMMITS}>
      <CommitsView
        commits={pullReqCommits?.commits || []}
        repoMetadata={repoMetadata}
        emptyTitle={getString('noCommits')}
        emptyMessage={getString('noCommitsPR')}
        pullRequestMetadata={pullReqMetadata}
        loading={!pullReqCommits?.commits}
        showHistoryIcon={true}
      />
    </PullRequestTabContentWrapper>
  )
}
