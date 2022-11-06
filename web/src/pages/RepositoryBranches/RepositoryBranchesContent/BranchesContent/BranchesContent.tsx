import React, { useMemo } from 'react'
import { Container, Color, TableV2 as Table, Text, Avatar, Tag } from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import { Link } from 'react-router-dom'
import Keywords from 'react-keywords'
import { orderBy } from 'lodash-es'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { RepoBranch, TypesRepository } from 'services/scm'
import { formatDate } from 'utils/Utils'
import css from './BranchesContent.module.scss'

interface BranchesContentProps {
  searchTerm?: string
  repoMetadata: TypesRepository
  branches: RepoBranch[]
}

export function BranchesContent({ repoMetadata, searchTerm = '', branches }: BranchesContentProps) {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const columns: Column<RepoBranch>[] = useMemo(
    () => [
      {
        Header: getString('branch'),
        width: '30%',
        Cell: ({ row }: CellProps<RepoBranch>) => {
          return (
            <Text className={css.rowText} color={Color.BLACK}>
              <Link
                to={routes.toSCMRepository({
                  repoPath: repoMetadata.path as string,
                  gitRef: row.original?.name
                })}
                className={css.commitLink}>
                <Keywords value={searchTerm}>{row.original?.name}</Keywords>
              </Link>
              {row.original?.name === repoMetadata.defaultBranch && (
                <>
                  <span className={css.spacer} />
                  <span className={css.spacer} />
                  <Tag>{getString('defaultBranch')}</Tag>
                </>
              )}
            </Text>
          )
        }
      },
      {
        Header: getString('status'),
        width: 'calc(70% - 200px)',
        Cell: ({ row }: CellProps<RepoBranch>) => {
          return (
            <Text color={Color.BLACK} lineClamp={1} className={css.rowText}>
              {/* TBD - Backend does not have information for branch status yet */}
            </Text>
          )
        }
      },
      {
        Header: getString('updated'),
        width: '200px',
        Cell: ({ row }: CellProps<RepoBranch>) => {
          return (
            <Text className={css.rowText} color={Color.BLACK}>
              <Avatar hoverCard={false} size="small" name={row.original.commit?.author?.identity?.name || ''} />
              <span className={css.spacer} />
              {formatDate(row.original.commit?.author?.when as string)}
            </Text>
          )
        }
      }
    ],
    [getString, repoMetadata.defaultBranch, repoMetadata.path, routes, searchTerm]
  )

  return (
    <Container className={css.container}>
      <Table<RepoBranch>
        className={css.table}
        columns={columns}
        data={orderBy(branches || [], [searchTerm ? '' : 'commit.author.when'], ['desc'])}
        getRowClassName={() => css.row}
      />
    </Container>
  )
}
