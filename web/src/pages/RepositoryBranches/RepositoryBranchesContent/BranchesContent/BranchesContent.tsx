import React, { useMemo } from 'react'
import { Container, Color, TableV2 as Table, Text, Avatar, Tag, Intent, useToaster } from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import { Link, useHistory } from 'react-router-dom'
import cx from 'classnames'
import Keywords from 'react-keywords'
import { useMutate } from 'restful-react'
import { String, useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { RepoBranch, TypesRepository } from 'services/scm'
import { formatDate, getErrorMessage } from 'utils/Utils'
import { useConfirmAction } from 'hooks/useConfirmAction'
import { MenuDivider, OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import css from './BranchesContent.module.scss'

interface BranchesContentProps {
  searchTerm?: string
  repoMetadata: TypesRepository
  branches: RepoBranch[]
  onDeleteSuccess: () => void
}

export function BranchesContent({ repoMetadata, searchTerm = '', branches, onDeleteSuccess }: BranchesContentProps) {
  const { routes } = useAppContext()
  const history = useHistory()
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
        width: 'calc(70% - 230px)',
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
      },
      {
        id: 'action',
        width: '30px',
        Cell: ({ row }: CellProps<RepoBranch>) => {
          const { mutate: deleteBranch } = useMutate({
            verb: 'DELETE',
            path: `/api/v1/repos/${repoMetadata.path}/+/branches/${row.original.name}`
          })
          const { showSuccess, showError } = useToaster()
          const confirmDeleteBranch = useConfirmAction({
            title: getString('deleteBranch'),
            confirmText: getString('delete'),
            intent: Intent.DANGER,
            message: <String useRichText stringID="deleteBranchConfirm" vars={{ name: row.original.name }} />,
            action: async () => {
              deleteBranch({})
                .then(() => {
                  showSuccess(getString('branchDeleted', { branch: row.original.name }), 5000)
                  onDeleteSuccess()
                })
                .catch(error => {
                  showError(getErrorMessage(error), 0, 'failedToDeleteBranch')
                })
            }
          })

          return (
            <OptionsMenuButton
              items={[
                {
                  icon: 'folder-close',
                  text: getString('browse'),
                  onClick: () => {
                    history.push(
                      routes.toSCMRepository({
                        repoPath: repoMetadata.path as string,
                        gitRef: row.original?.name
                      })
                    )
                  }
                },
                {
                  icon: 'comparison',
                  text: getString('compare'),
                  onClick: confirmDeleteBranch,
                  disabled: true
                },
                MenuDivider,
                {
                  icon: 'cross',
                  text: getString('delete'),
                  onClick: confirmDeleteBranch
                }
              ]}
            />
          )
        }
      }
    ],
    [getString, repoMetadata.defaultBranch, repoMetadata.path, routes, searchTerm, history, onDeleteSuccess]
  )

  return (
    <Container className={css.container}>
      <Table<RepoBranch>
        className={css.table}
        columns={columns}
        data={branches || []}
        getRowClassName={row => cx(css.row, (row.index + 1) % 2 ? css.odd : '')}
      />
    </Container>
  )
}
