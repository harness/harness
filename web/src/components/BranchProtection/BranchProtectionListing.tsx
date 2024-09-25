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
import {
  Container,
  Layout,
  Popover,
  Accordion,
  TableV2,
  Utils,
  Text,
  Button,
  ButtonVariation,
  Toggle,
  useToaster,
  FlexExpander,
  StringSubstitute
} from '@harnessio/uicore'
import cx from 'classnames'

import type { CellProps, Column } from 'react-table'
import { useGet, useMutate } from 'restful-react'
import { FontVariation } from '@harnessio/design-system'
import { Position } from '@blueprintjs/core'
import { useHistory } from 'react-router-dom'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useQueryParams } from 'hooks/useQueryParams'
import { usePageIndex } from 'hooks/usePageIndex'
import {
  getErrorMessage,
  LIST_FETCHING_LIMIT,
  permissionProps,
  type PageBrowserProps,
  Rule,
  RuleFields,
  BranchProtectionRulesMapType,
  createRuleFieldsMap
} from 'utils/Utils'
import { SettingTypeMode } from 'utils/GitUtils'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { useStrings } from 'framework/strings'

import { useConfirmAct } from 'hooks/useConfirmAction'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import type { OpenapiRule, ProtectionPattern } from 'services/code'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import Include from '../../icons/Include.svg?url'
import Exclude from '../../icons/Exclude.svg?url'
import BranchProtectionForm from './BranchProtectionForm/BranchProtectionForm'
import BranchProtectionHeader from './BranchProtectionHeader/BranchProtectionHeader'
import css from './BranchProtectionListing.module.scss'

const BranchProtectionListing = (props: { activeTab: string }) => {
  const { activeTab } = props
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const history = useHistory()
  const { routes } = useAppContext()
  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const [searchTerm, setSearchTerm] = useState('')
  const [curRuleName, setCurRuleName] = useState('')
  const { repoMetadata, settingSection, ruleId, settingSectionMode } = useGetRepositoryMetadata()
  const newRule = settingSection && settingSectionMode === SettingTypeMode.NEW
  const editRule = settingSection !== '' && ruleId !== '' && settingSectionMode === SettingTypeMode.EDIT
  const {
    data: rules,
    refetch: refetchRules,
    loading: loadingRules,
    response
  } = useGet<OpenapiRule[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/rules`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page,
      sort: 'date',
      order: 'desc',
      query: searchTerm
    },
    debounce: 500,
    lazy: !repoMetadata || !!editRule
  })

  const branchProtectionRules = {
    requireMinReviewersTitle: {
      title: getString('branchProtection.requireMinReviewersTitle'),
      requiredRule: {
        [RuleFields.APPROVALS_REQUIRE_MINIMUM_COUNT]: true
      }
    },
    reqReviewFromCodeOwnerTitle: {
      title: getString('branchProtection.reqReviewFromCodeOwnerTitle'),
      requiredRule: {
        [RuleFields.APPROVALS_REQUIRE_CODE_OWNERS]: true
      }
    },
    reqResOfChanges: {
      title: getString('branchProtection.reqResOfChanges'),
      requiredRule: {
        [RuleFields.APPROVALS_REQUIRE_NO_CHANGE_REQUEST]: true
      }
    },
    reqNewChangesTitle: {
      title: getString('branchProtection.reqNewChangesTitle'),
      requiredRule: {
        [RuleFields.APPROVALS_REQUIRE_LATEST_COMMIT]: true
      }
    },
    reqCommentResolutionTitle: {
      title: getString('branchProtection.reqCommentResolutionTitle'),
      requiredRule: {
        [RuleFields.COMMENTS_REQUIRE_RESOLVE_ALL]: true
      }
    },
    reqStatusChecksTitleAll: {
      title: getString('branchProtection.reqStatusChecksTitle'),
      requiredRule: {
        [RuleFields.STATUS_CHECKS_ALL_MUST_SUCCEED]: true
      }
    },
    reqStatusChecksTitle: {
      title: getString('branchProtection.reqStatusChecksTitle'),
      requiredRule: {
        [RuleFields.STATUS_CHECKS_REQUIRE_IDENTIFIERS]: true
      }
    },
    limitMergeStrategies: {
      title: getString('branchProtection.limitMergeStrategies'),
      requiredRule: {
        [RuleFields.MERGE_STRATEGIES_ALLOWED]: true
      }
    },
    autoDeleteTitle: {
      title: getString('branchProtection.autoDeleteTitle'),
      requiredRule: {
        [RuleFields.MERGE_DELETE_BRANCH]: true
      }
    },
    blockBranchCreation: {
      title: getString('branchProtection.blockBranchCreation'),
      requiredRule: {
        [RuleFields.LIFECYCLE_CREATE_FORBIDDEN]: true
      }
    },
    blockBranchDeletion: {
      title: getString('branchProtection.blockBranchDeletion'),
      requiredRule: {
        [RuleFields.LIFECYCLE_DELETE_FORBIDDEN]: true
      }
    },
    blockBranchUpdate: {
      title: getString('branchProtection.blockBranchUpdate'),
      requiredRule: {
        [RuleFields.MERGE_BLOCK]: true,
        [RuleFields.LIFECYCLE_UPDATE_FORBIDDEN]: true
      }
    },
    requirePr: {
      title: getString('branchProtection.requirePr'),
      requiredRule: {
        [RuleFields.LIFECYCLE_UPDATE_FORBIDDEN]: true,
        [RuleFields.MERGE_BLOCK]: false
      }
    },
    blockForcePush: {
      title: getString('branchProtection.blockForcePush'),
      requiredRule: {
        [RuleFields.LIFECYCLE_UPDATE_FORCE_FORBIDDEN]: true
      }
    }
  }

  const columns: Column<OpenapiRule>[] = useMemo(
    () => [
      {
        id: 'title',
        width: '100%',
        Cell: ({ row }: CellProps<OpenapiRule>) => {
          const [checked, setChecked] = useState<boolean>(
            row.original.state === 'active' || row.original.state === 'monitor' ? true : false
          )

          const { mutate } = useMutate<OpenapiRule>({
            verb: 'PATCH',
            path: `/api/v1/repos/${repoMetadata?.path}/+/rules/${row.original?.identifier}`
          })
          const [popoverDialogOpen, setPopoverDialogOpen] = useState(false)
          const { mutate: deleteRule } = useMutate({
            verb: 'DELETE',
            path: `/api/v1/repos/${repoMetadata?.path}/+/rules/${row.original.identifier}`
          })
          const confirmDelete = useConfirmAct()
          const includeElements = (row.original?.pattern as ProtectionPattern)?.include?.map(
            (includedString: string, index: number) => {
              return (
                <Container flex={{ align: 'center-center' }} className={css.greyButton} key={index}>
                  <img width={16} height={16} src={Include} />
                  <Text tooltipProps={{ popoverClassName: css.popover }} lineClamp={1}>
                    {includedString}
                  </Text>
                </Container>
              )
            }
          )
          const excludeElements = (row.original?.pattern as ProtectionPattern)?.exclude?.map(
            (excludedString: string, index: number) => {
              return (
                <Container flex={{ align: 'center-center' }} className={css.greyButton} key={index}>
                  <img width={16} height={16} src={Exclude} />
                  <Text tooltipProps={{ popoverClassName: css.popover }} lineClamp={1}>
                    {excludedString}
                  </Text>
                </Container>
              )
            }
          )
          const defaultElement = !!(row.original?.pattern as ProtectionPattern)?.default && (
            <Text flex={{ align: 'center-center' }} lineClamp={1} className={css.greyButton}>
              {getString('defaultBranch')}
            </Text>
          )

          type NonEmptyRule = {
            field: string // eslint-disable-next-line @typescript-eslint/no-explicit-any
            value: any
          }

          const checkAppliedRules = (rulesData: Rule, rulesList: BranchProtectionRulesMapType): NonEmptyRule[] => {
            const nonEmptyFields: NonEmptyRule[] = []
            const rulesDefinitionData: Record<RuleFields, boolean> = createRuleFieldsMap(rulesData)
            for (const [key, rule] of Object.entries(rulesList)) {
              const { title, requiredRule } = rule
              const isApplicable = Object.entries(requiredRule).every(([ruleField, requiredValue]) => {
                const ruleFieldEnum = ruleField as RuleFields
                const actualValue = rulesDefinitionData[ruleFieldEnum]
                if (requiredValue) return actualValue
                return !actualValue
              })
              if (isApplicable) {
                nonEmptyFields.push({
                  field: key,
                  value: title
                })
              }
            }

            return nonEmptyFields
          }

          const nonEmptyRules = checkAppliedRules(row.original.definition as Rule, branchProtectionRules)

          const { hooks, standalone } = useAppContext()

          const space = useGetSpaceParam()

          const permPushResult = hooks?.usePermissionTranslate?.(
            {
              resource: {
                resourceType: 'CODE_REPOSITORY',
                resourceIdentifier: repoMetadata?.identifier as string
              },
              permissions: ['code_repo_edit']
            },
            [space]
          )
          return (
            <Layout.Horizontal spacing="medium" padding={{ left: 'medium' }}>
              <Container onClick={Utils.stopEvent}>
                <Popover
                  isOpen={popoverDialogOpen && !permPushResult}
                  onInteraction={nextOpenState => {
                    setPopoverDialogOpen(nextOpenState)
                  }}
                  content={
                    <Container padding={'medium'} width={250}>
                      <Layout.Vertical spacing={'medium'}>
                        <Text font={{ variation: FontVariation.H5, size: 'medium' }}>
                          {checked
                            ? getString('branchProtection.disableTheRule')
                            : getString('branchProtection.enableTheRule')}
                        </Text>
                        <Text
                          margin={{ bottom: 'medium' }}
                          lineClamp={6}
                          font={{ variation: FontVariation.BODY2_SEMI }}>
                          <StringSubstitute
                            str={checked ? getString('disableWebhookContent') : getString('enableWebhookContent')}
                            vars={{
                              name: <strong>{row.original?.identifier}</strong>
                            }}
                          />
                        </Text>
                        <Layout.Horizontal spacing={'small'}>
                          <Button
                            variation={ButtonVariation.PRIMARY}
                            text={getString('confirm')}
                            onClick={() => {
                              const data = { state: checked ? 'disabled' : 'active' }
                              mutate(data)
                                .then(() => {
                                  showSuccess(getString('branchProtection.ruleUpdated'))
                                })
                                .catch(err => {
                                  showError(getErrorMessage(err))
                                })
                              setChecked(!checked)
                              setPopoverDialogOpen(false)
                            }}></Button>
                          <Container>
                            <Button
                              variation={ButtonVariation.TERTIARY}
                              text={getString('cancel')}
                              onClick={() => {
                                setPopoverDialogOpen(false)
                              }}></Button>
                          </Container>
                        </Layout.Horizontal>
                      </Layout.Vertical>
                    </Container>
                  }
                  position={Position.RIGHT}
                  interactionKind="click">
                  <Toggle
                    {...permissionProps(permPushResult, standalone)}
                    padding={{ top: 'xsmall' }}
                    key={`${row.original.identifier}-toggle`}
                    // className={cx(css.toggle, checked ? css.toggleEnable : css.toggleDisable)}
                    checked={checked}></Toggle>
                </Popover>
              </Container>
              <Container padding={{ left: 'small' }} style={{ flexGrow: 1 }}>
                <Layout.Horizontal spacing="small">
                  <Layout.Vertical>
                    <Text padding={{ right: 'small', top: 'xsmall' }} className={css.title}>
                      {row.original.identifier}
                    </Text>

                    {!!row.original.description && (
                      <Text
                        lineClamp={4}
                        width={'70vw'}
                        padding={{ right: 'small', top: 'medium' }}
                        className={css.text}>
                        {row.original.description}
                      </Text>
                    )}
                  </Layout.Vertical>
                  <FlexExpander />
                  <Container margin={{ left: 'medium' }} onClick={Utils.stopEvent}>
                    <OptionsMenuButton
                      width="100px"
                      isDark
                      {...permissionProps(permPushResult, standalone)}
                      items={[
                        {
                          hasIcon: true,
                          iconName: 'Edit',
                          text: getString('branchProtection.editRule'),
                          onClick: () => {
                            history.push(
                              routes.toCODESettings({
                                repoPath: repoMetadata?.path as string,
                                settingSection: settingSection,
                                settingSectionMode: SettingTypeMode.EDIT,
                                ruleId: String(row.original.identifier)
                              })
                            )
                          }
                        },
                        {
                          hasIcon: true,
                          iconName: 'main-trash',
                          text: getString('branchProtection.deleteRule'),
                          onClick: async () => {
                            confirmDelete({
                              className: css.hideButtonIcon,
                              title: getString('branchProtection.deleteProtectionRule'),
                              message: getString('branchProtection.deleteText', { rule: row.original.identifier }),
                              action: async () => {
                                try {
                                  await deleteRule({})
                                  showSuccess(getString('branchProtection.ruleDeleted'), 5000)
                                  setPage(1)
                                  refetchRules()
                                } catch (exception) {
                                  showError(getErrorMessage(exception), 0, 'failedToDeleteRule')
                                }
                              }
                            })
                          }
                        }
                      ]}
                    />
                  </Container>
                </Layout.Horizontal>
                <hr className={css.dividerContainer} />
                <Layout.Horizontal padding={{ top: 'xsmall', bottom: 'xsmall' }} spacing={'xsmall'}>
                  <Container>
                    <Text lineClamp={1} width={150} className={css.targetText}>
                      {getString('branchProtection.targetBranches')}
                    </Text>
                  </Container>
                  <Layout.Horizontal spacing="xsmall" className={css.widthContainer}>
                    {defaultElement}
                    {includeElements} {excludeElements}
                  </Layout.Horizontal>
                </Layout.Horizontal>
                <hr className={css.dividerContainer} />
                <Container onClick={Utils.stopEvent}>
                  <Accordion detailsClassName={cx({ [css.hideDetailsContainer]: nonEmptyRules.length === 0 })}>
                    <Accordion.Panel
                      disabled={nonEmptyRules.length === 0}
                      details={
                        nonEmptyRules.length > 0 ? (
                          <>
                            {nonEmptyRules.map((rule: { value: string }) => {
                              return (
                                <Text
                                  key={`${row.original.identifier}-${rule.value}`}
                                  className={css.appliedRulesTextContainer}>
                                  {rule.value}
                                </Text>
                              )
                            })}
                          </>
                        ) : (
                          ''
                        )
                      }
                      id="protectionApplied"
                      summary={`${nonEmptyRules.length} Rules applied`}
                    />
                  </Accordion>
                </Container>
              </Container>
            </Layout.Horizontal>
          )
        }
      }
    ], // eslint-disable-next-line react-hooks/exhaustive-deps
    [history, getString, repoMetadata?.path, setPage, showError, showSuccess]
  )
  const { hooks, standalone } = useAppContext()

  const space = useGetSpaceParam()

  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: ['code_repo_edit']
    },
    [space]
  )
  return (
    <Container>
      <LoadingSpinner visible={loadingRules} />
      {repoMetadata && !newRule && !editRule && (
        <BranchProtectionHeader
          activeTab={activeTab}
          onSearchTermChanged={(value: React.SetStateAction<string>) => {
            setSearchTerm(value)
            setPage(1)
          }}
          repoMetadata={repoMetadata}
        />
      )}
      {newRule || editRule ? (
        <BranchProtectionForm
          editMode={editRule}
          repoMetadata={repoMetadata}
          ruleUid={curRuleName}
          refetchRules={refetchRules}
          settingSectionMode={settingSectionMode}
        />
      ) : (
        <Container padding="xlarge">
          {!!rules && (
            <>
              <TableV2
                className={css.table}
                hideHeaders
                columns={columns}
                data={rules}
                getRowClassName={() => css.row}
                onRowClick={row => {
                  setCurRuleName(row.identifier as string)
                  history.push(
                    routes.toCODESettings({
                      repoPath: repoMetadata?.path as string,
                      settingSection: settingSection,
                      settingSectionMode: SettingTypeMode.EDIT,
                      ruleId: String(row.identifier)
                    })
                  )
                }}
              />

              <ResourceListingPagination response={response} page={page} setPage={setPage} />
            </>
          )}

          <NoResultCard
            showWhen={() => rules?.length === 0}
            forSearch={!!searchTerm}
            message={getString('branchProtection.ruleEmpty')}
            buttonText={getString('branchProtection.newRule')}
            onButtonClick={() => {
              history.push(
                routes.toCODESettings({
                  repoPath: repoMetadata?.path as string,
                  settingSection: activeTab,
                  settingSectionMode: SettingTypeMode.NEW
                })
              )
            }}
            permissionProp={permissionProps(permPushResult, standalone)}
          />
        </Container>
      )}
    </Container>
  )
}

export default BranchProtectionListing
