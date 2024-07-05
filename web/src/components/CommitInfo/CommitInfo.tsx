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

import {
  Button,
  Container,
  FlexExpander,
  Layout,
  Text,
  ButtonSize,
  ButtonVariation,
  Avatar,
  StringSubstitute
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import React, { useMemo } from 'react'
import { Link, useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { defaultTo } from 'lodash-es'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import type { TypesCommit, RepoRepositoryOutput } from 'services/code'
import type { GitInfoProps } from 'utils/GitUtils'
import type { CODERoutes } from 'RouteDefinitions'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { LIST_FETCHING_LIMIT } from 'utils/Utils'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { useDocumentTitle } from 'hooks/useDocumentTitle'
import css from './CommitInfo.module.scss'

const CommitInfo = (props: { repoMetadata: RepoRepositoryOutput; commitRef: string }) => {
  const { repoMetadata, commitRef } = props
  const history = useHistory()
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const { data: commits } = useGet<{ commits: TypesCommit[] }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      git_ref: commitRef || repoMetadata?.default_branch
    },
    lazy: !repoMetadata
  })
  const commitURL = routes.toCODECommit({
    repoPath: repoMetadata.path as string,
    commitRef: commitRef
  })
  const commitData = useMemo(
    () => commits?.commits?.filter(commit => commit.sha === commitRef)?.[0],
    [commitRef, commits?.commits]
  )
  function renderPullRequestLinkFromCommitMessage(
    repositoryMetadata: GitInfoProps['repoMetadata'],
    codeRoutes: CODERoutes,
    commitMessage = ''
  ) {
    let message: string | JSX.Element = commitMessage
    const match = message.match(/\(#\d+\)$/)

    if (match?.length) {
      message = message.replace(match[0], '({URL})')
      const pullRequestId = match[0].replace('(#', '').replace(')', '')

      message = (
        <StringSubstitute
          str={message}
          vars={{
            URL: (
              <Link
                to={codeRoutes.toCODEPullRequest({
                  repoPath: repositoryMetadata.path as string,
                  pullRequestId
                })}>
                #{pullRequestId}
              </Link>
            )
          }}
        />
      )
    }

    return message
  }

  useDocumentTitle(defaultTo(commitData?.title, getString('commit')))

  return (
    <>
      {commitData && (
        <Container className={css.commitInfoContainer} padding={{ top: 'small' }}>
          <Container className={css.commitTitleContainer} color={Color.GREY_100}>
            <Layout.Horizontal className={css.alignContent} padding={{ right: 'medium' }}>
              <Text
                className={css.titleText}
                icon={'code-commit'}
                iconProps={{ size: 16, margin: { right: 'small' } }}
                padding="medium"
                color="black">
                {defaultTo(renderPullRequestLinkFromCommitMessage(repoMetadata, routes, commitData?.title), '')}
              </Text>
              <FlexExpander />
              <Button
                className={css.btn}
                size={ButtonSize.SMALL}
                variation={ButtonVariation.SECONDARY}
                text={getString('browseFiles')}
                onClick={() => {
                  history.push(
                    routes.toCODERepository({
                      repoPath: repoMetadata.path as string,
                      gitRef: commitRef
                    })
                  )
                }}
              />
            </Layout.Horizontal>
          </Container>
          <Container className={css.infoContainer}>
            <Layout.Horizontal className={css.alignContent} padding={{ left: 'small', right: 'medium' }}>
              <Avatar hoverCard={false} size="small" name={defaultTo(commitData.author?.identity?.name, '')} />
              <Text className={css.infoText} color={Color.BLACK}>
                {defaultTo(commitData.author?.identity?.name, '')}
              </Text>
              <Text font={{ size: 'small' }} padding={{ left: 'small', top: 'medium', bottom: 'medium' }}>
                {getString('committed')}
                <TimePopoverWithLocal
                  padding={{ left: 'xsmall' }}
                  time={defaultTo(commitData?.committer?.when as unknown as number, 0)}
                  inline={false}
                  font={{ size: 'small' }}
                  color={Color.GREY_500}
                />
              </Text>

              <FlexExpander />
              <CommitActions sha={commitRef} href={commitURL} enableCopy />
            </Layout.Horizontal>
          </Container>
        </Container>
      )}
    </>
  )
}

export default CommitInfo
