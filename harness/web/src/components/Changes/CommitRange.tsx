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

import React, { useEffect, useMemo, useRef, useState } from 'react'
import { useHistory } from 'react-router-dom'
import { Render } from 'react-jsx-match'
import { useGet } from 'restful-react'
import { normalizeGitRef, type GitInfoProps } from 'utils/GitUtils'
import type { TypesCommit, TypesPullReq } from 'services/code'
import { PullRequestSection } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { useShowRequestError } from 'hooks/useShowRequestError'
import CommitRangeDropdown from './CommitRangeDropdown/CommitRangeDropdown'

interface CommitRangeProps extends Pick<GitInfoProps, 'repoMetadata'> {
  targetRef?: string
  sourceRef?: string
  pullRequestMetadata?: TypesPullReq
  defaultCommitRange?: string[]
  setCommitRange: React.Dispatch<React.SetStateAction<string[]>>
}

export const CommitRange: React.FC<CommitRangeProps> = ({
  repoMetadata,
  targetRef,
  sourceRef,
  pullRequestMetadata,
  setCommitRange: _setCommitRange,
  defaultCommitRange = []
}) => {
  const history = useHistory()
  const { routes } = useAppContext()
  const [commitRange, setCommitRange] = useState(defaultCommitRange)
  const { data: prCommits, error } = useGet<{
    commits: TypesCommit[]
  }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      git_ref: normalizeGitRef(sourceRef),
      after: normalizeGitRef(targetRef)
    },
    lazy: !pullRequestMetadata?.number
  })
  const commitSHA = useMemo(
    () =>
      commitRange.length === 0
        ? ''
        : commitRange.length === 1
        ? commitRange[0]
        : `${commitRange[0]}~1...${commitRange[commitRange.length - 1]}`,
    [commitRange]
  )
  const commitSHARef = useRef(commitSHA)

  useEffect(
    function updatePageWhenCommitRangeIsChanged() {
      if (!pullRequestMetadata?.number) {
        return
      }

      if (commitSHARef.current !== commitSHA) {
        commitSHARef.current = commitSHA

        history.push(
          routes.toCODEPullRequest({
            repoPath: repoMetadata.path as string,
            pullRequestId: String(pullRequestMetadata?.number),
            pullRequestSection: PullRequestSection.FILES_CHANGED,
            commitSHA
          })
        )
      }
    },
    [commitRange, history, routes, repoMetadata.path, pullRequestMetadata?.number, commitSHA]
  )

  useEffect(
    function updateCommitRangeForCaller() {
      _setCommitRange(commitRange)
    },
    [commitRange, _setCommitRange]
  )

  useShowRequestError(error)

  return (
    <Render when={pullRequestMetadata?.number}>
      <CommitRangeDropdown
        allCommits={prCommits?.commits || []}
        selectedCommits={commitRange}
        setSelectedCommits={setCommitRange}
      />
    </Render>
  )
}
