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
import React, { useCallback, useMemo, useState } from 'react'
import {
  Container,
  Layout,
  Popover,
  TableV2,
  Utils,
  Text,
  Button,
  ButtonVariation,
  Toggle,
  useToaster,
  FlexExpander,
  StringSubstitute,
  SelectOption,
  PageBody
} from '@harnessio/uicore'
import cx from 'classnames'
import type { CellProps, Column } from 'react-table'
import { useGet, useMutate } from 'restful-react'
import { capitalize, debounce, isEmpty } from 'lodash-es'
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
  getScopeData,
  getEditPermissionRequestFromScope,
  getEditPermissionRequestFromIdentifier,
  ScopeEnum,
  OrderSortDate
} from 'utils/Utils'
import { ProtectionRulesType, RulesTargetType, SettingTypeMode } from 'utils/GitUtils'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { useStrings } from 'framework/strings'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import type { OpenapiRule, ProtectionPattern, RepoRepositoryOutput } from 'services/code'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useGetCurrentPageScope } from 'hooks/useGetCurrentPageScope'
import { getConfig } from 'services/config'
import type { CODEProps } from 'RouteDefinitions'
import ProtectionRulesForm from './ProtectionRulesForm/ProtectionRulesForm'
import ProtectionRulesHeader from './ProtectionRulesHeader/ProtectionRulesHeader'
import Include from '../../icons/Include.svg?url'
import Exclude from '../../icons/Exclude.svg?url'
import {
  createRuleFieldsMap,
  getProtectionRules,
  ProtectionRulesMapType,
  Rule,
  RuleFields,
  RuleState
} from './ProtectionRulesUtils'
import css from './ProtectionRulesListing.module.scss'

const getPatterns = (pattern: ProtectionPattern, included: boolean) => {
  return (pattern as ProtectionPattern)?.[included ? RulesTargetType.INCLUDE : RulesTargetType.EXCLUDE]?.map(
    (ref: string, index: number) => (
      <Container flex={{ align: 'center-center' }} className={css.greyButton} key={index}>
        <img width={12} height={12} src={included ? Include : Exclude} />
        <Text className={css.pattern} tooltipProps={{ popoverClassName: css.popover }} lineClamp={1}>
          {ref}
        </Text>
      </Container>
    )
  )
}

const ProtectionRulesListing = (props: { activeTab: string; repoMetadata?: RepoRepositoryOutput }) => {
  const { activeTab, repoMetadata } = props
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const space = useGetSpaceParam()
  const history = useHistory()
  const { routes, standalone, hooks } = useAppContext()
  const pageBrowser = useQueryParams<PageBrowserProps & { type: ProtectionRulesType }>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const [searchTerm, setSearchTerm] = useState<string | undefined>()
  const [debouncedSearchTerm, setDebouncedSearchTerm] = useState(searchTerm)
  const { settingSection, ruleId, settingSectionMode } = useParams<CODEProps>()
  const newRule = settingSection && settingSectionMode === SettingTypeMode.NEW
  const editRule = !isEmpty(settingSection) && !isEmpty(ruleId) && settingSectionMode === SettingTypeMode.EDIT
  const [inheritRules, setInheritRules] = useState<boolean>(false)
  const currentPageScope = useGetCurrentPageScope()

  const getRulesPath = useMemo(
    () =>
      currentPageScope === ScopeEnum.REPO_SCOPE ? `/repos/${repoMetadata?.path}/+/rules` : `/spaces/${space}/+/rules`,
    [currentPageScope, repoMetadata?.path, space]
  )

  const {
    data: rules,
    refetch: refetchRules,
    loading: loadingRules,
    error,
    response
  } = useGet<OpenapiRule[]>({
    base: getConfig('code/api/v1'),
    path: getRulesPath,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      inherited: inheritRules,
      page,
      sort: 'date',
      order: OrderSortDate.DESC,
      query: debouncedSearchTerm,
      type: pageBrowser.type ?? ''
    },
    lazy: !!editRule
  })

  const debouncedRefetch = useCallback(
    debounce((value: string) => {
      setDebouncedSearchTerm(value)
      setPage(1)
    }, 500),
    []
  )

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

    if (metaData && currentScope === ScopeEnum.REPO_SCOPE) {
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
          const { definition, description, identifier, pattern, scope, state, type } = row.original
          const { scopeRef, scopeIcon } = getScopeData(space, scope ?? 1, standalone)
          const getRuleIDPath = () =>
            scope === ScopeEnum.REPO_SCOPE && repoMetadata
              ? `/repos/${repoMetadata?.path}/+/rules/${encodeURIComponent(identifier as string)}`
              : `/spaces/${scopeRef}/+/rules/${encodeURIComponent(identifier as string)}`

          const [popoverDialogOpen, setPopoverDialogOpen] = useState(false)
          const [checked, setChecked] = useState<boolean>(
            [RuleState.ACTIVE, RuleState.MONITOR].includes(state as RuleState)
          )

          const { mutate: toggleRule } = useMutate<OpenapiRule>({
            verb: 'PATCH',
            base: getConfig('code/api/v1'),
            path: getRuleIDPath()
          })
          const { mutate: deleteRule } = useMutate({
            verb: 'DELETE',
            base: getConfig('code/api/v1'),
            path: getRuleIDPath()
          })
          const confirmDelete = useConfirmAct()

          const noPatternApplied = isEmpty(Object.keys(pattern || {}))
          const includedPatterns = getPatterns(pattern as ProtectionPattern, true)
          const excludedPatterns = getPatterns(pattern as ProtectionPattern, false)
          const defaultElement = !!(pattern as ProtectionPattern)?.default && (
            <Text flex={{ align: 'center-center' }} lineClamp={1} className={cx(css.greyButton, css.pattern)}>
              {getString('defaultBranch')}
            </Text>
          )

          const checkAppliedRules = (rulesData: Rule, rulesList: ProtectionRulesMapType): SelectOption[] => {
            const nonEmptyFields: SelectOption[] = []
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
                  label: key,
                  value: title
                })
              }
            }

            return nonEmptyFields
          }

          const nonEmptyRules = checkAppliedRules(definition as Rule, getProtectionRules(getString, type))

          const permPushResult = hooks?.usePermissionTranslate(
            getEditPermissionRequestFromScope(space, scope ?? 0, repoMetadata),
            [space, repoMetadata, scope]
          )

          return (
            <Layout.Vertical spacing="small" padding={{ left: 'medium', bottom: 'xsmall' }}>
              <Layout.Horizontal flex={{ align: 'center-center' }}>
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
                              ? getString('protectionRules.disableTheRule')
                              : getString('protectionRules.enableTheRule')}
                          </Text>
                          <Text margin={{ bottom: 'medium' }} font={{ variation: FontVariation.BODY2_SEMI }}>
                            <StringSubstitute
                              str={checked ? getString('disableWebhookContent') : getString('enableWebhookContent')}
                              vars={{
                                name: <strong>{identifier}</strong>
                              }}
                            />
                          </Text>
                          <Layout.Horizontal spacing={'small'}>
                            <Button
                              variation={ButtonVariation.PRIMARY}
                              text={getString('confirm')}
                              onClick={() => {
                                const data = { state: checked ? RuleState.DISABLED : RuleState.ACTIVE }
                                toggleRule(data)
                                  .then(() => {
                                    showSuccess(
                                      getString('protectionRules.ruleUpdated', { ruleType: capitalize(type) })
                                    )
                                  })
                                  .catch(err => {
                                    showError(getErrorMessage(err))
                                  })
                                setChecked(!checked)
                                setPopoverDialogOpen(false)
                              }}
                            />
                            <Container>
                              <Button
                                variation={ButtonVariation.TERTIARY}
                                text={getString('cancel')}
                                onClick={() => {
                                  setPopoverDialogOpen(false)
                                }}
                              />
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
                      key={`${identifier}-toggle`}
                      checked={checked}
                    />
                  </Popover>
                </Container>
                <Layout.Horizontal flex={{ align: 'center-center' }} padding={{ top: 'xsmall', left: 'small' }}>
                  {inheritRules && <Icon padding={{ right: 'small', bottom: 'xsmall' }} name={scopeIcon} size={16} />}
                  <Text className={css.title}>{identifier}</Text>
                </Layout.Horizontal>
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
                        text: getString('protectionRules.editRule'),
                        onClick: () => {
                          navigateToSettings({
                            repoMetadata,
                            standalone,
                            space,
                            scope,
                            settingSection,
                            settingSectionMode: SettingTypeMode.EDIT,
                            ruleId: String(identifier)
                          })
                        }
                      },
                      {
                        hasIcon: true,
                        iconName: 'main-trash',
                        text: getString('protectionRules.deleteRule'),
                        onClick: async () => {
                          confirmDelete({
                            className: css.hideButtonIcon,
                            title: getString('protectionRules.deleteProtectionRule'),
                            message: getString('protectionRules.deleteText', { rule: identifier }),
                            action: async () => {
                              try {
                                await deleteRule({})
                                showSuccess(getString('protectionRules.ruleDeleted'), 5000)
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

              {!!description && (
                <Text lineClamp={3} className={css.text}>
                  {description}
                </Text>
              )}

              {!noPatternApplied && (
                <Layout.Horizontal
                  spacing="xsmall"
                  className={css.widthContainer}
                  padding={{ top: description ? 'small' : 'xsmall' }}>
                  {defaultElement} {includedPatterns} {excludedPatterns}
                </Layout.Horizontal>
              )}

              {!isEmpty(nonEmptyRules) ? (
                <Layout.Horizontal
                  padding={{ top: 'small' }}
                  className={cx({
                    [css.appliedRulesContainer]: !noPatternApplied
                  })}>
                  <Container>
                    <Text className={css.rulesText}>
                      {getString('protectionRules.numberOfRulesApplied', {
                        count: nonEmptyRules.length,
                        type,
                        rulesLabel: (nonEmptyRules.length || 0) > 1 ? 'rules' : 'rule'
                      })}
                    </Text>
                  </Container>
                  <Container className={css.rulesContainer} flex>
                    {nonEmptyRules.map((rule, idx) => (
                      <>
                        <Text key={rule.value as string} className={css.appliedRulesTextContainer}>
                          {rule.value}
                        </Text>
                        {idx !== nonEmptyRules.length - 1 && <div className={css.divider} />}
                      </>
                    ))}
                  </Container>
                </Layout.Horizontal>
              ) : null}
            </Layout.Vertical>
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
    <PageBody loading={loadingRules} error={error} retryOnError={() => refetchRules()} className={css.pageBody}>
      {!newRule && !editRule && (
        <ProtectionRulesHeader
          activeTab={activeTab}
          currentPageScope={currentPageScope}
          onSearchTermChanged={(value: string) => {
            setSearchTerm(value)
            debouncedRefetch(value)
          }}
          inheritRules={inheritRules}
          setInheritRules={setInheritRules}
          ruleTypeFilter={pageBrowser.type}
          {...(repoMetadata && { repoMetadata: repoMetadata })}
        />
      )}
      {newRule || editRule ? (
        <ProtectionRulesForm
          currentPageScope={currentPageScope}
          editMode={editRule}
          repoMetadata={repoMetadata}
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
            showWhen={() => isEmpty(rules) && !loadingRules}
            forSearch={!!searchTerm}
            message={getString('protectionRules.ruleEmpty')}
            buttonText={getString('protectionRules.newRule')}
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
    </PageBody>
  )
}

export default ProtectionRulesListing
