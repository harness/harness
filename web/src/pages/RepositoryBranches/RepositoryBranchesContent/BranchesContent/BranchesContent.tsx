import React, { useEffect, useMemo, useState } from 'react'
import { Container, Color, TableV2 as Table, Text, Avatar, Tag, Intent, useToaster } from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import { Link, useHistory } from 'react-router-dom'
import cx from 'classnames'
import Keywords from 'react-keywords'
import { useMutate } from 'restful-react'
import { String, useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type {
  OpenapiCalculateCommitDivergenceRequest,
  RepoBranch,
  RepoCommitDivergence,
  TypesRepository
} from 'services/code'
import { formatDate, getErrorMessage } from 'utils/Utils'
import { useConfirmAction } from 'hooks/useConfirmAction'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { CommitDivergence } from 'components/CommitDivergence/CommitDivergence'
import { makeDiffRefs } from 'utils/GitUtils'
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
  const { mutate: getBranchDivergence } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/commits/calculate-divergence`
  })
  const [divergence, setDivergence] = useState<RepoCommitDivergence[]>([])
  const branchDivergenceRequestBody: OpenapiCalculateCommitDivergenceRequest = useMemo(() => {
    return {
      maxCount: 0,
      requests: branches?.map(branch => ({ from: branch.name, to: repoMetadata.default_branch }))
    }
  }, [repoMetadata, branches])

  useEffect(() => {
    if (branchDivergenceRequestBody.requests?.length) {
      setDivergence([])
      getBranchDivergence(branchDivergenceRequestBody).then((response: RepoCommitDivergence[]) => {
        setDivergence(response)
      })
    }
  }, [getBranchDivergence, branchDivergenceRequestBody])

  const columns: Column<RepoBranch>[] = useMemo(
    () => [
      {
        Header: getString('branch'),
        width: '30%',
        Cell: ({ row }: CellProps<RepoBranch>) => {
          return (
            <Text
              lineClamp={1}
              className={cx(css.rowText, row.original?.name === repoMetadata.default_branch ? css.defaultBranch : '')}
              color={Color.BLACK}>
              <Link
                to={routes.toCODERepository({
                  repoPath: repoMetadata.path as string,
                  gitRef: row.original?.name
                })}
                className={css.commitLink}>
                <Keywords value={searchTerm}>{row.original?.name}</Keywords>
              </Link>
            </Text>
          )
        }
      },
      {
        Header: getString('status'),
        Id: 'status',
        width: 'calc(70% - 230px)',
        Cell: ({ row }: CellProps<RepoBranch>) => {
          if (row.original?.name === repoMetadata.default_branch) {
            return (
              <Container flex={{ align: 'center-center' }} width={150}>
                <Tag>{getString('defaultBranch')}</Tag>
              </Container>
            )
          }

          return (
            <CommitDivergence
              defaultBranch={repoMetadata.default_branch as string}
              behind={divergence?.[row.index]?.behind as number}
              ahead={divergence?.[row.index]?.ahead as number}
            />
          )
        }
      },
      {
        Header: getString('updated'),
        width: '200px',
        Cell: ({ row }: CellProps<RepoBranch>) => {
          return (
            <Text lineClamp={1} className={css.rowText} color={Color.BLACK} tag="div">
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
              isDark
              width="100px"
              items={[
                {
                  text: getString('browse'),
                  onClick: () => {
                    history.push(
                      routes.toCODERepository({
                        repoPath: repoMetadata.path as string,
                        gitRef: row.original?.name
                      })
                    )
                  }
                },
                {
                  text: getString('compare'),
                  onClick: () => {
                    history.push(
                      routes.toCODECompare({
                        repoPath: repoMetadata.path as string,
                        diffRefs: makeDiffRefs(repoMetadata.default_branch as string, row.original?.name as string)
                      })
                    )
                  }
                },
                '-',
                {
                  text: getString('delete'),
                  isDanger: true,
                  onClick: confirmDeleteBranch
                }
              ]}
            />
          )
        }
      }
    ],
    [
      getString,
      repoMetadata.default_branch,
      repoMetadata.path,
      routes,
      searchTerm,
      history,
      onDeleteSuccess,
      divergence
    ]
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
