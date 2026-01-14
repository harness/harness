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

import React, { useMemo } from 'react'
import {
  Container,
  TableV2 as Table,
  Text,
  Avatar,
  Layout,
  ButtonVariation,
  Button,
  FlexExpander,
  StringSubstitute,
  Popover
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import type { CellProps, Column } from 'react-table'
import { defaultTo } from 'lodash-es'
import { Link, useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { TypesCommit } from 'services/code'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { ThreadSection } from 'components/ThreadSection/ThreadSection'
import { FileSection, formatDate, PullRequestSection } from 'utils/Utils'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import type { CODERoutes } from 'RouteDefinitions'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import css from './CommitsView.module.scss'

interface CommitsViewProps extends Pick<GitInfoProps, 'repoMetadata'> {
  commits: TypesCommit[] | null
  emptyTitle: string
  emptyMessage: string
  showFileHistoryIcons?: boolean
  showHistoryIcon?: boolean
  resourcePath?: string
  setActiveTab?: React.Dispatch<React.SetStateAction<string>>
  pullRequestMetadata?: GitInfoProps['pullReqMetadata']
  loading?: boolean
}

export function CommitsView({
  repoMetadata,
  commits,
  emptyTitle,
  emptyMessage,
  showFileHistoryIcons = false,
  showHistoryIcon = false,
  resourcePath = '',
  setActiveTab,
  pullRequestMetadata,
  loading
}: CommitsViewProps) {
  const history = useHistory()
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const columns: Column<TypesCommit>[] = useMemo(
    () => [
      {
        id: 'author',
        width: '20%',
        Cell: ({ row }: CellProps<TypesCommit>) => {
          return (
            <Layout.Horizontal
              spacing="small"
              flex={{ alignItems: 'center' }}
              style={{
                display: 'inline-flex'
              }}>
              <Avatar hoverCard={false} size="small" name={row.original.author?.identity?.name || ''} />
              <Text lineClamp={1} padding={{ left: 'large' }} className={css.rowText} color={Color.BLACK}>
                {row.original.author?.identity?.name}
              </Text>
            </Layout.Horizontal>
          )
        }
      },
      {
        id: 'timestamp',
        width: '20%',
        Cell: ({ row }: CellProps<TypesCommit>) => {
          return (
            <Text
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_500}
              padding={{ left: 'xsmall' }}
              style={{ display: 'block', flexWrap: 'nowrap' }}>
              {getString('committed')}
              <TimePopoverWithLocal
                padding={{ left: 'xsmall' }}
                time={defaultTo(row.original.committer?.when as unknown as number, 0)}
                inline={false}
                font={{ variation: FontVariation.SMALL }}
                color={Color.GREY_500}
              />
            </Text>
          )
        }
      },
      {
        id: 'commit',
        width: 'calc(60% - 100px)',
        Cell: ({ row }: CellProps<TypesCommit>) => {
          return (
            <Text
              tooltipProps={{ popoverClassName: css.popover }}
              color={Color.BLACK}
              lineClamp={1}
              className={css.rowText}>
              {renderPullRequestLinkFromCommitMessage(repoMetadata, routes, row.original.message)}
            </Text>
          )
        }
      },
      {
        id: 'sha',
        width: '100px',
        Cell: ({ row }: CellProps<TypesCommit>) => {
          return (
            <CommitActions
              sha={row.original.sha as string}
              href={
                pullRequestMetadata?.number
                  ? routes.toCODEPullRequest({
                      repoPath: repoMetadata.path as string,
                      pullRequestId: String(pullRequestMetadata.number),
                      pullRequestSection: PullRequestSection.FILES_CHANGED,
                      commitSHA: row.original.sha
                    })
                  : routes.toCODECommit({
                      repoPath: repoMetadata.path as string,
                      commitRef: row.original.sha as string
                    })
              }
              enableCopy
            />
          )
        }
      },
      {
        id: 'buttons',
        width: showFileHistoryIcons || showHistoryIcon ? '60px' : '0px',
        Cell: ({ row }: CellProps<TypesCommit>) => {
          if (showFileHistoryIcons) {
            return (
              <Container padding={{ left: 'small' }}>
                <Layout.Horizontal className={css.layout}>
                  <Popover
                    content={
                      <Text color={Color.BLACK} padding="medium">
                        {getString('viewFile')}
                      </Text>
                    }
                    interactionKind="hover">
                    <Icon
                      id={css.commitFileButton}
                      className={css.fileButton}
                      name={'code-content'}
                      size={14}
                      onClick={() => {
                        history.push(
                          routes.toCODERepository({
                            repoPath: repoMetadata.path as string,
                            gitRef: row.original.sha,
                            resourcePath
                          })
                        )
                        if (setActiveTab) {
                          setActiveTab(FileSection.CONTENT)
                        }
                      }}
                    />
                  </Popover>
                  <Button
                    id={css.commitRepoButton}
                    variation={ButtonVariation.ICON}
                    text={'<>'}
                    onClick={() => {
                      history.push(
                        routes.toCODERepository({
                          repoPath: repoMetadata.path as string,
                          gitRef: row.original.sha
                        })
                      )
                    }}
                    tooltip={getString('viewRepo')}
                  />
                </Layout.Horizontal>
              </Container>
            )
          } else if (showHistoryIcon) {
            return (
              <Container>
                <Layout.Horizontal className={css.historyBtnLayout}>
                  <Button
                    id={css.commitRepoButton}
                    variation={ButtonVariation.ICON}
                    text={'<>'}
                    onClick={() => {
                      history.push(
                        routes.toCODERepository({
                          repoPath: repoMetadata.path as string,
                          gitRef: row.original.sha
                        })
                      )
                    }}
                    tooltip={getString('viewRepo')}
                  />
                </Layout.Horizontal>
              </Container>
            )
          } else {
            return <Container width={0}></Container>
          }
        }
      }
    ],
    [repoMetadata, routes] // eslint-disable-line react-hooks/exhaustive-deps
  )
  const commitsGroupedByDate: Record<string, TypesCommit[]> = useMemo(
    () =>
      commits?.reduce((group, commit) => {
        const date = formatDate(commit.committer?.when as string)
        group[date] = (group[date] || []).concat(commit)
        return group
      }, {} as Record<string, TypesCommit[]>) || {},
    [commits]
  )

  return (
    <Container className={css.container}>
      <Layout.Horizontal>
        <FlexExpander />
      </Layout.Horizontal>
      {!!commits?.length &&
        Object.entries(commitsGroupedByDate).map(([date, commitsByDate]) => {
          return (
            <ThreadSection
              key={date}
              title={
                <Text
                  padding={{ top: 'small', bottom: 'small' }}
                  icon={CodeIcon.Commit}
                  iconProps={{ size: 20 }}
                  color={Color.GREY_500}
                  className={css.label}>
                  {getString('commitsOn', { date })}
                </Text>
              }>
              <Table<TypesCommit>
                className={css.table}
                hideHeaders
                columns={columns}
                data={commitsByDate || []}
                getRowClassName={() => css.row}
              />
            </ThreadSection>
          )
        })}
      <NoResultCard
        showWhen={() => commits?.length === 0 && !loading}
        forSearch={true}
        title={emptyTitle}
        emptySearchMessage={emptyMessage}
      />
    </Container>
  )
}

function renderPullRequestLinkFromCommitMessage(
  repoMetadata: GitInfoProps['repoMetadata'],
  routes: CODERoutes,
  commitMessage = ''
) {
  let message: string | JSX.Element = commitMessage
  const match = message.match(/\(#\d+\)(\n|$)/)
  if (match?.length) {
    message = message.replace(match[0], '({URL})')
    const pullRequestId = match[0].replace('(#', '').replace(')', '').replace('\n', '')
    message = (
      <StringSubstitute
        str={message}
        vars={{
          URL: (
            <Link
              to={routes.toCODEPullRequest({
                repoPath: repoMetadata.path as string,
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
