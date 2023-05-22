import React, { useEffect, useMemo, useState } from 'react'
import {
  Container,
  Color,
  TableV2 as Table,
  Text,
  Avatar,
  Intent,
  useToaster,
} from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import { Link, useHistory } from 'react-router-dom'
import cx from 'classnames'
import Keywords from 'react-keywords'
import { useMutate } from 'restful-react'
import { noop } from 'lodash-es'
import { String, useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'

import type {
  OpenapiCalculateCommitDivergenceRequest,
  RepoBranch,
  RepoCommitDivergence,
  TypesRepository
} from 'services/code'
import { formatDate, getErrorMessage, voidFn } from 'utils/Utils'
import { useConfirmAction } from 'hooks/useConfirmAction'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { useCreateBranchModal } from 'components/CreateBranchModal/CreateBranchModal'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { CodeIcon } from 'utils/GitUtils'
import css from './TagsContent.module.scss'

interface TagsContentProps {
  searchTerm?: string
  repoMetadata: TypesRepository
  branches: RepoBranch[]
  onDeleteSuccess: () => void
}

export function TagsContent({ repoMetadata, searchTerm = '', branches, onDeleteSuccess }: TagsContentProps) {
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
  const onSuccess = voidFn(noop)
  const openModal = useCreateBranchModal({ repoMetadata, onSuccess })

  const columns: Column<RepoBranch>[] = useMemo(
    () => [
      {
        Header: getString('tag'),
        width: '50%',
        Cell: ({ row }: CellProps<RepoBranch>) => {
          return (
            <Text
              icon="code-tag"
              iconProps={{ size: 22, color: Color.GREY_300 }}
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
        Header: getString('commit'),
        Id: 'commit',
        width: '20%',
        Cell: ({ row }: CellProps<RepoBranch>) => {
          return (
            <CommitActions
              sha={row.original.sha as string}
              href={routes.toCODECommits({
                repoPath: repoMetadata.path as string,
                commitRef: row.original.sha as string
              })}
              enableCopy
            />
          )
        }
      },

      {
        Header: getString('tagger'),
        width: '20%',
        Cell: ({ row }: CellProps<RepoBranch>) => {
          return (
            <Text className={css.rowText} color={Color.BLACK} tag="div">
              <Avatar hoverCard={false} size="small" name={row.original.commit?.author?.identity?.name || ''} />
              <span className={css.spacer} />
              {row.original.commit?.author?.identity?.name || ''}
            </Text>
          )
        }
      },
      {
        Header: getString('creationDate'),
        width: '200px',
        Cell: ({ row }: CellProps<RepoBranch>) => {
          return (
            <Text className={css.rowText} color={Color.BLACK} tag="div">
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
            path: `/api/v1/repos/${repoMetadata.path}/+/tags/${row.original.name}`
          })
          const { showSuccess, showError } = useToaster()
          const confirmDeleteTag = useConfirmAction({
            title: getString('deleteTag'),
            confirmText: getString('confirmDelete'),
            intent: Intent.DANGER,
            message: <String useRichText stringID="deleteTagConfirm" vars={{ name: row.original.name }} />,
            action: async () => {
              deleteBranch({})
                .then(() => {
                  showSuccess(getString('tagDeleted', { branch: row.original.name }), 5000)
                  onDeleteSuccess()
                })
                .catch(error => {
                  showError(getErrorMessage(error), 0, 'failedToDeleteTag')
                })
            }
          })
          return (
            <OptionsMenuButton
              width="100px"
              items={[
                {
                  text: getString('createBranch'),
                  iconName: CodeIcon.BranchSmall,
                  hasIcon: true,
                  iconSize: 16,
                  onClick: () => {
                    openModal()
                  }
                },
                {
                  text: getString('viewFiles'),
                  iconName: CodeIcon.FileLight,
                  iconSize: 16,
                  hasIcon: true,
                  onClick: () => {
                    history.push(
                      routes.toCODERepository({
                        repoPath: repoMetadata.path as string,
                        gitRef: row.original?.name
                      })
                    )
                  }
                },
                '-',
                {
                  text: getString('deleteTag'),
                  iconName: CodeIcon.Delete,
                  iconSize: 16,
                  hasIcon: true,
                  isDanger: true,
                  onClick: confirmDeleteTag
                }
              ]}
              isDark
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
