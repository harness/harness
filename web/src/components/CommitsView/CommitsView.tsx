import React, { useMemo } from 'react'
import {
  Container,
  TableV2 as Table,
  Text,
  Avatar,
  Layout,
  ButtonVariation,
  ButtonSize,
  Button,
  FlexExpander,
  StringSubstitute,
  Popover
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import type { CellProps, Column } from 'react-table'
import { noop, orderBy } from 'lodash-es'
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
import css from './CommitsView.module.scss'

interface CommitsViewProps extends Pick<GitInfoProps, 'repoMetadata'> {
  commits: TypesCommit[] | null
  emptyTitle: string
  emptyMessage: string
  prStatsChanged?: Number
  handleRefresh?: () => void
  showFileHistoryIcons?: boolean
  resourcePath?: string
  setActiveTab?: React.Dispatch<React.SetStateAction<string>>
  pullRequestMetadata?: GitInfoProps['pullRequestMetadata']
}

export function CommitsView({
  repoMetadata,
  commits,
  emptyTitle,
  emptyMessage,
  handleRefresh = noop,
  prStatsChanged,
  showFileHistoryIcons = false,
  resourcePath = '',
  setActiveTab,
  pullRequestMetadata
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
            <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }} style={{ display: 'inline-flex' }}>
              <Avatar hoverCard={false} size="small" name={row.original.author?.identity?.name || ''} />
              <Text
                lineClamp={1}
                padding={{ right: 'small' }}
                icon="code-tag"
                iconProps={{ size: 22 }}
                className={css.rowText}
                color={Color.BLACK}>
                {row.original.author?.identity?.name}
              </Text>
            </Layout.Horizontal>
          )
        }
      },
      {
        id: 'commit',
        width: 'calc(80% - 100px)',
        Cell: ({ row }: CellProps<TypesCommit>) => {
          return (
            <Text color={Color.BLACK} lineClamp={1} className={css.rowText}>
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
        width: showFileHistoryIcons ? '60px' : '0px',
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
        {!prStatsChanged ? null : (
          <Button
            onClick={handleRefresh}
            iconProps={{ className: css.refreshIcon, size: 12 }}
            icon="repeat"
            text={getString('refresh')}
            variation={ButtonVariation.SECONDARY}
            size={ButtonSize.SMALL}
            margin={{ bottom: 'small' }}
          />
        )}
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
                data={orderBy(commitsByDate || [], ['committer.when'], ['desc'])}
                getRowClassName={() => css.row}
              />
            </ThreadSection>
          )
        })}
      <NoResultCard
        showWhen={() => commits?.length === 0}
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
