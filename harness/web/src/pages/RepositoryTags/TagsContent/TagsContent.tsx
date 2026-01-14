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

import React, { useMemo, useState } from 'react'
import { Container, TableV2 as Table, Text, Avatar, useToaster, StringSubstitute, Layout } from '@harnessio/uicore'
import { Color, FontVariation, Intent } from '@harnessio/design-system'
import type { CellProps, Column } from 'react-table'
import { Link, useHistory } from 'react-router-dom'
import cx from 'classnames'
import Keywords from 'react-keywords'
import { useMutate } from 'restful-react'
import { isEmpty, noop } from 'lodash-es'
import { Icon } from '@harnessio/icons'
import { Render } from 'react-jsx-match'
import { String, useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { TypesBranchExtended, TypesCommitTag, RepoRepositoryOutput } from 'services/code'
import { formatDate, getErrorMessage, voidFn } from 'utils/Utils'
import { useConfirmAction } from 'hooks/useConfirmAction'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { useCreateBranchModal } from 'components/CreateRefModal/CreateBranchModal/CreateBranchModal'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { CodeIcon, GitRefType, REFS_TAGS_PREFIX } from 'utils/GitUtils'
import { useRuleViolationCheck } from 'hooks/useRuleViolationCheck'
import css from './TagsContent.module.scss'

interface TagsContentProps {
  searchTerm?: string
  repoMetadata: RepoRepositoryOutput
  branches: TypesBranchExtended[]
  onDeleteSuccess: () => void
}

export function TagsContent({ repoMetadata, searchTerm = '', branches, onDeleteSuccess }: TagsContentProps) {
  const { routes } = useAppContext()
  const history = useHistory()
  const { getString } = useStrings()

  const onSuccess = voidFn(noop)

  const columns: Column<TypesBranchExtended>[] = useMemo(
    () => [
      {
        Header: getString('tag'),
        width: '20%',
        Cell: ({ row }: CellProps<TypesCommitTag>) => {
          return (
            <Text
              icon="code-tag"
              lineClamp={1}
              width={`100%`}
              tooltipProps={{ popoverClassName: css.popover }}
              iconProps={{ size: 22, color: Color.GREY_300 }}
              className={cx(css.rowText, row.original?.name === repoMetadata.default_branch ? css.defaultBranch : '')}
              color={Color.BLACK}>
              <Link
                to={routes.toCODERepository({
                  repoPath: repoMetadata.path as string,
                  gitRef: `${REFS_TAGS_PREFIX}${row.original?.name}`
                })}
                className={css.commitLink}>
                <Keywords value={searchTerm}>{row.original?.name}</Keywords>
              </Link>
            </Text>
          )
        }
      },
      {
        Header: getString('description'),
        width: '35%',
        Cell: ({ row }: CellProps<TypesCommitTag>) => {
          return (
            <Text className={cx(css.rowText)} color={Color.BLACK} lineClamp={1} width={`100%`}>
              {row.original?.message}
            </Text>
          )
        }
      },
      {
        Header: getString('commit'),
        Id: 'commit',
        width: '15%',
        Cell: ({ row }: CellProps<TypesCommitTag>) => {
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
        Header: getString('tagger'),
        width: '15%',
        Cell: ({ row }: CellProps<TypesCommitTag>) => {
          return (
            <Text lineClamp={1} className={css.rowText} color={Color.BLACK} tag="div">
              {row.original.tagger?.identity?.name ? (
                <Avatar hoverCard={false} size="small" name={row.original.tagger?.identity?.name || ''} />
              ) : (
                ''
              )}
              <span className={css.spacer} />
              {row.original.tagger?.identity?.name || ''}
            </Text>
          )
        }
      },
      {
        Header: getString('creationDate'),
        width: '200px',
        Cell: ({ row }: CellProps<TypesCommitTag>) => {
          return row.original.tagger?.when ? (
            <Text className={css.rowText} color={Color.BLACK} tag="div">
              <span className={css.spacer} />
              {formatDate(row.original.tagger?.when as string)}
            </Text>
          ) : (
            ''
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
          const { mutate: deleteTag } = useMutate({
            verb: 'DELETE',
            path: `/api/v1/repos/${repoMetadata.path}/+/tags/${row.original.name}`,
            queryParams: { dry_run_rules: dryRun, bypass_rules: bypassed }
          })
          const { showSuccess, showError } = useToaster()
          const confirmDeleteTag = useConfirmAction({
            title: getString('deleteTag'),
            confirmText:
              !dryRun && (!violation || !bypassable)
                ? getString('delete')
                : getString('protectionRules.deleteRefAlertBtn', { ref: GitRefType.TAG }),
            buttonDisabled: !dryRun && !bypassable,
            intent: Intent.DANGER,
            message: <String useRichText stringID="deleteTagConfirm" vars={{ name: row.original.name }} />,
            persistDialog,
            onOpen: () => {
              deleteTag({})
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
                  showError(getErrorMessage(error), 0, 'deleteTagDryRunFailed')
                })
                .finally(() => setDryRun(false))
            },
            action: async () => {
              deleteTag({})
                .then(() => {
                  showSuccess(
                    <StringSubstitute
                      str={getString('tagDeleted')}
                      vars={{
                        tag: row.original.name
                      }}
                    />,
                    5000
                  )
                  onDeleteSuccess()
                })
                .catch(error => {
                  showError(getErrorMessage(error), 0, 'failedToDeleteTag')
                })
                .finally(() => setDryRun(false))
            },
            childtag: (
              <Render when={violation}>
                <Layout.Horizontal className={css.warningMessage}>
                  <Icon intent={Intent.WARNING} name="danger-icon" size={16} />
                  <Text font={{ variation: FontVariation.BODY2 }} color={Color.RED_800}>
                    {bypassable
                      ? getString('protectionRules.deleteRefAlertText', { ref: GitRefType.TAG })
                      : getString('protectionRules.deleteRefBlockText', { ref: GitRefType.TAG })}
                  </Text>
                </Layout.Horizontal>
              </Render>
            )
          })
          const openModal = useCreateBranchModal({
            repoMetadata,
            onSuccess,
            showSuccessMessage: true,
            suggestedSourceBranch: row.original.name,
            showBranchTag: false,
            refIsATag: true
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
                        gitRef: `${REFS_TAGS_PREFIX}${row.original?.name}`
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
      // eslint-disable-line react-hooks/exhaustive-deps
      getString,
      routes,
      searchTerm,
      history,
      onDeleteSuccess,
      repoMetadata,
      onSuccess
    ] // eslint-disable-line react-hooks/exhaustive-deps
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
