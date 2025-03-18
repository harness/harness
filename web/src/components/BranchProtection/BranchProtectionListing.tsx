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
import { useHistory, useParams } from 'react-router-dom'
import { Icon } from '@harnessio/icons'
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
  createRuleFieldsMap,
  LabelsPageScope,
  getScopeData,
  getScopeIcon,
  getEditPermissionRequestFromScope,
  getEditPermissionRequestFromIdentifier
} from 'utils/Utils'
import { SettingTypeMode } from 'utils/GitUtils'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { useStrings } from 'framework/strings'

import { useConfirmAct } from 'hooks/useConfirmAction'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import type { OpenapiRule, ProtectionPattern, RepoRepositoryOutput } from 'services/code'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import type { CODEProps } from 'RouteDefinitions'
import { getConfig } from 'services/config'
import Include from '../../icons/Include.svg?url'
import Exclude from '../../icons/Exclude.svg?url'
import BranchProtectionForm from './BranchProtectionForm/BranchProtectionForm'
import BranchProtectionHeader from './BranchProtectionHeader/BranchProtectionHeader'
import css from './BranchProtectionListing.module.scss'

const BranchProtectionListing = (props: {
  activeTab: string
  repoMetadata?: RepoRepositoryOutput
  currentPageScope: LabelsPageScope
}) => {
  const { activeTab, repoMetadata, currentPageScope } = props
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const history = useHistory()
  const { routes, standalone, hooks } = useAppContext()
  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const [searchTerm, setSearchTerm] = useState('')
  const [currentRule, setCurrentRule] = useState<OpenapiRule>()
  const { settingSection, ruleId, settingSectionMode } = useParams<CODEProps>()
  const newRule = settingSection && settingSectionMode === SettingTypeMode.NEW
  const editRule = settingSection !== '' && ruleId !== '' && settingSectionMode === SettingTypeMode.EDIT
  const [showParentScopeFilter, setShowParentScopeFilter] = useState<boolean>(true)
  const [inheritRules, setInheritRules] = useState<boolean>(false)
  const space = useGetSpaceParam()

  useEffect(() => {
    if (currentPageScope) {
      if (currentPageScope === LabelsPageScope.ACCOUNT) setShowParentScopeFilter(false)
      else if (currentPageScope === LabelsPageScope.SPACE) setShowParentScopeFilter(false)
    }
  }, [currentPageScope, standalone])

  const getRulesPath = () =>
    currentPageScope === LabelsPageScope.REPOSITORY
      ? `/repos/${repoMetadata?.path}/+/rules`
      : `/spaces/${space}/+/rules`

  const {
    data: rules,
    refetch: refetchRules,
    loading: loadingRules,
    response
  } = useGet<OpenapiRule[]>({
    base: getConfig('code/api/v1'),
    path: getRulesPath(),
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      inherited: inheritRules,
      page,
      sort: 'date',
      order: 'desc',
      query: searchTerm
    },
    debounce: 500,
    lazy: !!editRule
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
    },
    autoAddCodeownersToReview: {
      title: getString('branchProtection.addCodeownersToReviewTitle'),
      requiredRule: {
        [RuleFields.AUTO_ADD_CODE_OWNERS]: true
      }
    },
    requireMinDefaultReviewersTitle: {
      title: getString('branchProtection.requireMinDefaultReviewersTitle'),
      requiredRule: {
        [RuleFields.APPROVALS_REQUIRE_MINIMUM_DEFAULT_REVIEWERS]: true
      }
    },
    defaultReviewersAdded: {
      title: getString('branchProtection.enableDefaultReviewersTitle'),
      requiredRule: {
        [RuleFields.DEFAULT_REVIEWERS_ADDED]: true
      }
    }
  }

  function navigateToSettings({
    repoMetadata: metaData,
    standalone: isStandalone,
    space: currentSpace,
    scope: currentScope,
    settingSection: section,
    settingSectionMode: sectionMode,
    ruleId: id
  }: {
    repoMetadata?: RepoRepositoryOutput
    standalone?: boolean
    space: string
    scope?: number
    settingSection?: string
    settingSectionMode?: string
    ruleId?: string
  }) {
    const { scopeRef } = currentScope
      ? getScopeData(currentSpace, currentScope, isStandalone ?? false)
      : { scopeRef: currentSpace }

    if (metaData && currentScope === 0) {
      history.push(
        routes.toCODESettings({
          repoPath: metaData.path as string,
          settingSection: section,
          settingSectionMode: sectionMode,
          ruleId: id
        })
      )
    } else if (isStandalone) {
      history.push(
        routes.toCODESpaceSettings({
          space: currentSpace,
          settingSection: section,
          settingSectionMode: sectionMode,
          ruleId: id
        })
      )
    } else {
      history.push(
        routes.toCODEManageRepositories({
          space: scopeRef,
          settingSection: section,
          settingSectionMode: sectionMode,
          ruleId: id
        })
      )
    }
  }

  const columns: Column<OpenapiRule>[] = useMemo(
    () => [
      {
        id: 'title',
        width: '100%',
        Cell: ({ row }: CellProps<OpenapiRule>) => {
          const { scopeRef } = getScopeData(space, row.original?.scope ?? 1, standalone)
          const getRuleIDPath = () =>
            row.original?.scope === 0 && repoMetadata
              ? `/repos/${repoMetadata?.path}/+/rules/${encodeURIComponent(row.original?.identifier as string)}`
              : `/spaces/${scopeRef}/+/rules/${encodeURIComponent(row.original?.identifier as string)}`

          const [checked, setChecked] = useState<boolean>(
            row.original.state === 'active' || row.original.state === 'monitor' ? true : false
          )

          const { mutate: toggleRule } = useMutate<OpenapiRule>({
            verb: 'PATCH',
            base: getConfig('code/api/v1'),
            path: getRuleIDPath()
          })
          const [popoverDialogOpen, setPopoverDialogOpen] = useState(false)
          const { mutate: deleteRule } = useMutate({
            verb: 'DELETE',
            base: getConfig('code/api/v1'),
            path: getRuleIDPath()
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

          const scope = row.original?.scope
          const permPushResult = hooks?.usePermissionTranslate(
            getEditPermissionRequestFromScope(space, scope ?? 0, repoMetadata),
            [space, repoMetadata, scope]
          )

          const scopeIcon = getScopeIcon(row.original?.scope, standalone)
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
                              toggleRule(data)
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
                    <Layout.Horizontal
                      padding={{ right: 'small', top: 'xsmall' }}
                      flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                      {scopeIcon && <Icon padding={{ right: 'small' }} name={scopeIcon} size={16} />}
                      <Text className={css.title}>{row.original.identifier}</Text>
                    </Layout.Horizontal>
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
                            setCurrentRule(row.original)
                            navigateToSettings({
                              repoMetadata,
                              standalone,
                              space,
                              scope,
                              settingSection,
                              settingSectionMode: SettingTypeMode.EDIT,
                              ruleId: String(row.original.identifier)
                            })
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

  const permPushResult = hooks?.usePermissionTranslate(getEditPermissionRequestFromIdentifier(space, repoMetadata), [
    space,
    repoMetadata
  ])
  return (
    <Container>
      <LoadingSpinner visible={loadingRules} />
      {!newRule && !editRule && (
        <BranchProtectionHeader
          activeTab={activeTab}
          showParentScopeFilter={showParentScopeFilter}
          onSearchTermChanged={(value: React.SetStateAction<string>) => {
            setSearchTerm(value)
            setPage(1)
          }}
          inheritRules={inheritRules}
          setInheritRules={setInheritRules}
          {...(repoMetadata && { repoMetadata: repoMetadata })}
        />
      )}
      {newRule || editRule ? (
        <BranchProtectionForm
          editMode={editRule}
          repoMetadata={repoMetadata}
          currentRule={currentRule}
          refetchRules={refetchRules}
          settingSectionMode={settingSectionMode}
          currentPageScope={currentPageScope}
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
                  setCurrentRule(row)
                  navigateToSettings({
                    repoMetadata,
                    standalone,
                    space,
                    scope: row.scope,
                    settingSection,
                    settingSectionMode: SettingTypeMode.EDIT,
                    ruleId: String(row.identifier)
                  })
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
              navigateToSettings({
                repoMetadata,
                standalone,
                space,
                settingSection,
                settingSectionMode: SettingTypeMode.NEW
              })
            }}
            permissionProp={permissionProps(permPushResult, standalone)}
          />
        </Container>
      )}
    </Container>
  )
}

export default BranchProtectionListing
