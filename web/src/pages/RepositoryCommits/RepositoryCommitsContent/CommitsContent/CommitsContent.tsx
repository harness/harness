import React, { useMemo } from 'react'
import { Container, Color, TableV2 as Table, Text, Avatar, Layout } from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import { orderBy } from 'lodash-es'
import { useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { RepoCommit, TypesRepository } from 'services/scm'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { formatDate } from 'utils/Utils'
import { GitIcon } from 'utils/GitUtils'
import css from './CommitsContent.module.scss'

interface CommitsContentProps {
  repoMetadata: TypesRepository
  contentInfo: RepoCommit[]
}

export function CommitsContent({ repoMetadata, contentInfo }: CommitsContentProps): JSX.Element {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const columns: Column<RepoCommit>[] = useMemo(
    () => [
      {
        Header: getString('name'),
        width: '20%',
        Cell: ({ row }: CellProps<RepoCommit>) => {
          return (
            <Text className={css.rowText} color={Color.BLACK}>
              <Avatar size="small" name={row.original.author?.identity?.name || ''} />
              <span className={css.spacer} />
              {row.original.author?.identity?.name}
            </Text>
          )
        }
      },
      {
        Header: getString('commits'),
        width: 'calc(80% - 100px)',
        Cell: ({ row }: CellProps<RepoCommit>) => {
          return (
            <Text color={Color.BLACK} lineClamp={1} className={css.rowText}>
              {row.original.message}
            </Text>
          )
        }
      },
      {
        Header: getString('repos.lastChange'),
        width: '100px',
        Cell: ({ row }: CellProps<RepoCommit>) => {
          return (
            <CommitActions
              sha={row.original.sha as string}
              href={routes.toSCMRepositoryCommits({
                repoPath: repoMetadata.path as string,
                commitRef: row.original.sha as string
              })}
              enableCopy
            />
          )
        }
      }
    ],
    [getString, repoMetadata.path, routes]
  )
  const commitsGroupedByDate: Record<string, RepoCommit[]> = useMemo(() => {
    const _commits: Record<string, RepoCommit[]> = {}
    contentInfo.forEach(commitInfo => {
      const date = formatDate(commitInfo.author?.when as string)
      _commits[date] = _commits[date] || []
      _commits[date].push(commitInfo)
    })
    return _commits
  }, [contentInfo])

  return (
    <Container className={css.container}>
      {Object.entries(commitsGroupedByDate).map(([date, commits]) => {
        return (
          <Container key={date} className={css.commitSection}>
            <Layout.Vertical spacing="medium">
              <Text icon={GitIcon.COMMIT} color={Color.GREY_500} className={css.label}>
                {getString('commitsOn', { date })}
              </Text>
              <Container className={css.commitTableContainer}>
                <Table<RepoCommit>
                  className={css.table}
                  hideHeaders
                  columns={columns}
                  data={orderBy(commits || [], ['author.when'], ['desc'])}
                  getRowClassName={() => css.row}
                />
              </Container>
            </Layout.Vertical>
          </Container>
        )
      })}
    </Container>
  )
}
