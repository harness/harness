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

import React, { useEffect, useMemo, useState } from 'react'
import {
  Container,
  TableV2 as Table,
  Layout,
  Text,
  Avatar,
  Tag,
  useToaster,
  StringSubstitute,
  useIsMounted
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { isEmpty, noop } from 'lodash-es'
import { Color, Intent, FontVariation } from '@harnessio/design-system'
import { Render } from 'react-jsx-match'
import type { CellProps, Column } from 'react-table'
import { Link, useHistory } from 'react-router-dom'
import cx from 'classnames'
import Keywords from 'react-keywords'
import { useMutate } from 'restful-react'
import { String, useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type {
  OpenapiCalculateCommitDivergenceRequest,
  TypesBranchExtended,
  TypesCommitDivergence,
  RepoRepositoryOutput
} from 'services/code'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { formatDate, getErrorMessage } from 'utils/Utils'
import { useConfirmAction } from 'hooks/useConfirmAction'
import { useRuleViolationCheck } from 'hooks/useRuleViolationCheck'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { CommitDivergence } from 'components/CommitDivergence/CommitDivergence'
import { GitRefType, makeDiffRefs, normalizeGitRef } from 'utils/GitUtils'
import css from './BranchesContent.module.scss'

interface BranchesContentProps {
  searchTerm?: string
  repoMetadata: RepoRepositoryOutput
  branches: TypesBranchExtended[]
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
  const [divergence, setDivergence] = useState<TypesCommitDivergence[]>([])
  const branchDivergenceRequestBody: OpenapiCalculateCommitDivergenceRequest = useMemo(() => {
    return {
      maxCount: 0,
      requests: branches?.map(branch => ({
        from: normalizeGitRef(branch.name),
        to: normalizeGitRef(repoMetadata.default_branch)
      }))
    }
  }, [repoMetadata, branches])
  const isMounted = useIsMounted()

  useEffect(() => {
    if (isMounted.current && branchDivergenceRequestBody.requests?.length) {
      setDivergence([])
      getBranchDivergence(branchDivergenceRequestBody)
        .then((response: TypesCommitDivergence[]) => {
          if (isMounted.current) {
            setDivergence(response)
          }
        })
        .catch(noop)
    }
  }, [getBranchDivergence, branchDivergenceRequestBody, isMounted])

  const columns: Column<TypesBranchExtended>[] = useMemo(
    () => [
      {
        Header: getString('branch'),
        width: '30%',
        Cell: ({ row }: CellProps<TypesBranchExtended>) => {
          return (
            <Text
              lineClamp={1}
              className={cx(css.rowText, row.original?.name === repoMetadata.default_branch ? css.defaultBranch : '')}
              color={Color.BLACK}
              tooltipProps={{
                popoverClassName: css.popover
              }}>
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
        Cell: ({ row }: CellProps<TypesBranchExtended>) => {
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
        Header: getString('commit'),
        Id: 'commit',
        width: '15%',
        Cell: ({ row }: CellProps<TypesBranchExtended>) => {
          return (
            <CommitActions
              sha={row.original.commit?.sha as string}
              href={routes.toCODECommit({
                repoPath: repoMetadata.path as string,
                commitRef: row.original.commit?.sha as string
              })}
              enableCopy
            />
          )
        }
      },
      {
        Header: getString('updated'),
        width: '200px',
        Cell: ({ row }: CellProps<TypesBranchExtended>) => {
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
        Cell: ({ row }: CellProps<TypesBranchExtended>) => {
          const { violation, bypassable, bypassed, setAllStates } = useRuleViolationCheck()
          const [persistDialog, setPersistDialog] = useState(true)
          const [dryRun, setDryRun] = useState(true)
          const { mutate: deleteBranch } = useMutate({
            verb: 'DELETE',
            path: `/api/v1/repos/${repoMetadata.path}/+/branches/${row.original.name}`,
            queryParams: { dry_run_rules: dryRun, bypass_rules: bypassed }
          })
          const { showSuccess, showError } = useToaster()
          const confirmDeleteBranch = useConfirmAction({
            title: getString('deleteBranch'),
            confirmText:
              !dryRun && (!violation || !bypassable)
                ? getString('delete')
                : getString('protectionRules.deleteRefAlertBtn', { ref: GitRefType.BRANCH }),
            buttonDisabled: !dryRun && !bypassable,
            intent: Intent.DANGER,
            message: <String useRichText stringID="deleteBranchConfirm" vars={{ name: row.original.name }} />,
            persistDialog,
            onOpen: () => {
              deleteBranch({})
                .then(res => {
                  if (!isEmpty(res?.rule_violations)) {
                    setAllStates({
                      violation: true,
                      bypassed: true,
                      bypassable: res?.rule_violations[0]?.bypassable
                    })
                  } else setAllStates({ bypassable: true })
                })
                .catch(error => {
                  setPersistDialog(false)
                  showError(getErrorMessage(error), 0, 'deleteBranchDryRunFailed')
                })
                .finally(() => setDryRun(false))
            },
            action: async () => {
              deleteBranch({})
                .then(() => {
                  showSuccess(
                    <StringSubstitute
                      str={getString('branchDeleted')}
                      vars={{
                        branch: row.original.name
                      }}
                    />,
                    5000
                  )
                  onDeleteSuccess()
                })
                .catch(error => {
                  showError(getErrorMessage(error), 0, 'failedToDeleteBranch')
                })
                .finally(() => setDryRun(false))
            },
            childtag: (
              <Render when={violation}>
                <Layout.Horizontal className={css.warningMessage}>
                  <Icon intent={Intent.WARNING} name="danger-icon" size={16} />
                  <Text font={{ variation: FontVariation.BODY2 }} color={Color.RED_800}>
                    {bypassable
                      ? getString('protectionRules.deleteRefAlertText', { ref: GitRefType.BRANCH })
                      : getString('protectionRules.deleteRefBlockText', { ref: GitRefType.BRANCH })}
                  </Text>
                </Layout.Horizontal>
              </Render>
            )
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
      <Table<TypesBranchExtended>
        className={css.table}
        columns={columns}
        data={branches || []}
        getRowClassName={row => cx(css.row, (row.index + 1) % 2 ? css.odd : '')}
      />
    </Container>
  )
}
