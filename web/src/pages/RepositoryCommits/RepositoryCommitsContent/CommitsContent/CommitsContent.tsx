import React, { useMemo } from 'react'
import { Container, Color, TableV2 as Table, Text, Avatar, Layout } from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import { orderBy } from 'lodash-es'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { RepoCommit, TypesRepository } from 'services/scm'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { formatDate } from 'utils/Utils'
import { GitIcon } from 'utils/GitUtils'
import css from './CommitsContent.module.scss'

interface CommitsContentProps {
  repoMetadata: TypesRepository
  commits: RepoCommit[]
}

export function CommitsContent({ repoMetadata, commits }: CommitsContentProps) {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const columns: Column<RepoCommit>[] = useMemo(
    () => [
      {
        id: 'author',
        width: '20%',
        Cell: ({ row }: CellProps<RepoCommit>) => {
          return (
            <Text className={css.rowText} color={Color.BLACK}>
              <Avatar hoverCard={false} size="small" name={row.original.author?.identity?.name || ''} />
              <span className={css.spacer} />
              {row.original.author?.identity?.name}
            </Text>
          )
        }
      },
      {
        id: 'commit',
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
        id: 'sha',
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
    [repoMetadata.path, routes]
  )
  const commitsGroupedByDate: Record<string, RepoCommit[]> = useMemo(
    () =>
      commits?.reduce((group, commit) => {
        const date = formatDate(commit.author?.when as string)
        group[date] = (group[date] || []).concat(commit)
        return group
      }, {} as Record<string, RepoCommit[]>) || {},
    [commits]
  )

  return (
    <Container className={css.container}>
      {Object.entries(commitsGroupedByDate).map(([date, commitsByDate]) => {
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
                  data={orderBy(commitsByDate || [], ['author.when'], ['desc'])}
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
