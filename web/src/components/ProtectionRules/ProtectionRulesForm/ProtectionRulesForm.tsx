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
import cx from 'classnames'
import * as yup from 'yup'
import {
  Avatar,
  Button,
  ButtonVariation,
  Container,
  FlexExpander,
  FormInput,
  Formik,
  FormikForm,
  Layout,
  SelectOption,
  SplitButton,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { Menu, PopoverPosition } from '@blueprintjs/core'
import { Icon } from '@harnessio/icons'
import { useHistory, useParams } from 'react-router-dom'
import { useGet, useMutate } from 'restful-react'
import { capitalize, isEmpty } from 'lodash-es'
import { useStrings } from 'framework/strings'
import {
  CodeIcon,
  MergeStrategy,
  ProtectionRulesType,
  RulesTargetType,
  SettingTypeMode,
  SettingsTab
} from 'utils/GitUtils'
import {
  ScopeEnum,
  REGEX_VALID_REPO_NAME,
  getEditPermissionRequestFromScope,
  getErrorMessage,
  getScopeData,
  getScopeFromParams,
  permissionProps,
  PrincipalType,
  INCLUDE_INHERITED_GROUPS
} from 'utils/Utils'
import type {
  RepoRepositoryOutput,
  OpenapiRule,
  TypesPrincipalInfo,
  ProtectionBranch,
  ProtectionTag
} from 'services/code'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { getConfig } from 'services/config'
import type { Identifier } from 'utils/types'
import SearchDropDown from 'components/SearchDropDown/SearchDropDown'
import { useQueryParams } from 'hooks/useQueryParams'
import BranchRulesForm from './components/BranchRulesForm'
import BypassList from './components/BypassList'
import Include from '../../../icons/Include.svg?url'
import Exclude from '../../../icons/Exclude.svg?url'
import TagRulesForm from './components/TagRulesForm'
import {
  combineAndNormalizePrincipalsAndGroups,
  getPayload,
  NormalizedPrincipal,
  rulesFormInitialPayload,
  RulesFormPayload,
  RuleState,
  transformDataToArray
} from '../ProtectionRulesUtils'
import css from './ProtectionRulesForm.module.scss'

const ProtectionRulesForm = (props: {
  currentPageScope: ScopeEnum
  editMode: boolean
  settingSectionMode: SettingTypeMode
  repoMetadata?: RepoRepositoryOutput
}) => {
  const { routes, routingId, standalone, hooks } = useAppContext()
  const params = useParams<Identifier>()
  const queryParams = useQueryParams<{ type: ProtectionRulesType }>()
  const { ruleId } = useGetRepositoryMetadata()
  const { showError, showSuccess } = useToaster()
  const space = useGetSpaceParam()
  const history = useHistory()
  const { currentPageScope, editMode = false, repoMetadata, settingSectionMode } = props
  const { getString } = useStrings()
  const [searchTerm, setSearchTerm] = useState('')
  const [searchStatusTerm, setSearchStatusTerm] = useState('')
  const [targetType, setTargetType] = useState(RulesTargetType.INCLUDE)
  const currentRuleScope = getScopeFromParams(params, standalone, repoMetadata)
  const { scopeRef } =
    typeof currentRuleScope === 'number' ? getScopeData(space, currentRuleScope, standalone) : { scopeRef: space }
  const [accountIdentifier, orgIdentifier, projectIdentifier] = scopeRef?.split('/') || []

  const getUpdateRulePath = () =>
    currentPageScope === ScopeEnum.REPO_SCOPE
      ? `/repos/${repoMetadata?.path}/+/rules/${encodeURIComponent(ruleId)}`
      : `/spaces/${scopeRef}/+/rules/${encodeURIComponent(ruleId)}`

  const getCreateRulePath = () =>
    currentPageScope === ScopeEnum.REPO_SCOPE ? `/repos/${repoMetadata?.path}/+/rules` : `/spaces/${space}/+/rules`

  const { data: rule } = useGet<OpenapiRule>({
    base: getConfig('code/api/v1'),
    path: getUpdateRulePath(),
    lazy: !ruleId
  })

  const { mutate } = useMutate({
    verb: 'POST',
    base: getConfig('code/api/v1'),
    path: getCreateRulePath()
  })

  const { mutate: updateRule } = useMutate({
    verb: 'PATCH',
    base: getConfig('code/api/v1'),
    path: getUpdateRulePath()
  })

  const ruleType = queryParams?.type || rule?.type || ProtectionRulesType.BRANCH

  const { data: principals, loading: loadingPrincipals } = useGet<TypesPrincipalInfo[]>({
    path: `/api/v1/principals`,
    queryParams: {
      query: searchTerm,
      type: standalone ? PrincipalType.USER : [PrincipalType.USER, PrincipalType.SERVICE_ACCOUNT],
      ...(!standalone && { inherited: true }),
      accountIdentifier: accountIdentifier || routingId,
      orgIdentifier,
      projectIdentifier
    },
    queryParamStringifyOptions: {
      arrayFormat: 'repeat'
    },
    debounce: 500
  })

  const { data: userGroups, loading: loadingUsersGroups } = useGet<any>({
    path: `/api/v1/usergroups`,
    queryParams: {
      query: searchTerm,
      filterType: INCLUDE_INHERITED_GROUPS,
      accountIdentifier: accountIdentifier || routingId,
      orgIdentifier,
      projectIdentifier
    },
    debounce: 500,
    lazy: standalone
  })

  const combinedOptions = useMemo(
    () => combineAndNormalizePrincipalsAndGroups(principals, userGroups),
    [principals, userGroups]
  )

  const getRecentChecksPath = () =>
    currentRuleScope === ScopeEnum.REPO_SCOPE && repoMetadata
      ? `/repos/${repoMetadata?.path}/+/checks/recent`
      : `/spaces/${scopeRef}/+/checks/recent`
  const { data: statuses } = useGet<string[]>({
    base: getConfig('code/api/v1'),
    path: getRecentChecksPath(),
    queryParams: {
      query: searchStatusTerm,
      ...(!repoMetadata && {
        recursive: true
      })
    },
    debounce: 500,
    lazy: ruleType !== ProtectionRulesType.BRANCH
  })

  const statusOptions: SelectOption[] = useMemo(
    () =>
      statuses?.map(status => ({
        value: status,
        label: status
      })) || [],
    [statuses]
  )

  // userPrincipalOptions used in default reviewers
  const userPrincipalOptions: SelectOption[] = useMemo(
    () =>
      principals?.reduce<SelectOption[]>((acc, principal) => {
        if (principal?.type === PrincipalType.USER) {
          const { id, uid, display_name, email } = principal
          acc.push({
            value: `${id?.toString()} ${uid}`,
            label: `${display_name} (${email})`
          })
        }
        return acc
      }, []) || [],
    [principals]
  )

  const getBypassListOptions = useCallback(
    (currentBypassList?: NormalizedPrincipal[]): SelectOption[] =>
      combinedOptions
        ?.filter(item => !currentBypassList?.some(principal => principal.id === item.id))
        .map(principal => ({
          value: JSON.stringify(principal),
          label: `${principal.display_name} (${principal.email_or_identifier})`
        })) || [],
    [combinedOptions]
  )

  const handleSubmit = async (operation: Promise<OpenapiRule>, successMessage: string, resetForm: () => void) => {
    try {
      await operation
      showSuccess(successMessage)
      resetForm()
      history.push(
        repoMetadata
          ? routes.toCODESettings({
              repoPath: repoMetadata?.path as string,
              settingSection: SettingsTab.PROTECTION_RULES
            })
          : standalone
          ? routes.toCODESpaceSettings({
              space,
              settingSection: SettingsTab.PROTECTION_RULES
            })
          : routes.toCODEManageRepositories({
              space,
              settingSection: SettingsTab.PROTECTION_RULES
            })
      )
    } catch (exception) {
      showError(getErrorMessage(exception))
    }
  }

  const initialValues = useMemo((): RulesFormPayload => {
    if (editMode && rule) {
      const {
        definition,
        description,
        identifier,
        pattern,
        state,
        users: usersMap,
        user_groups: userGroupsMap
      } = rule || {}
      const { bypass } = definition || {}

      const bypassListUsers = bypass?.user_ids?.map((id: number) => usersMap?.[id])
      const bypassUserGroups = bypass?.user_group_ids?.map(id => userGroupsMap?.[id])
      const transformedBypassList = transformDataToArray(bypassListUsers || [])
      const transformedUserGroupsBypassList = transformDataToArray(bypassUserGroups || [])
      const bypassList = combineAndNormalizePrincipalsAndGroups(transformedBypassList, transformedUserGroupsBypassList)

      // Create a new array based on the "include" key from the JSON object and the strings array
      const includeArr = pattern?.include?.map((arr: string) => [RulesTargetType.INCLUDE, arr]) ?? []
      const excludeArr = pattern?.exclude?.map((arr: string) => [RulesTargetType.EXCLUDE, arr]) ?? []

      const commonRulesForm = {
        name: identifier,
        desc: description,
        enable: state !== RuleState.DISABLED,
        target: '',
        targetDefault: pattern?.default,
        targetList: [...includeArr, ...excludeArr],
        allRepoOwners: bypass?.repo_owners,
        bypassList: bypassList,
        targetSet: false,
        bypassSet: false
      }

      switch (ruleType) {
        case ProtectionRulesType.BRANCH: {
          const { lifecycle, pullreq } = (definition as ProtectionBranch) || {}

          const defaultReviewersUsers = pullreq?.reviewers?.default_reviewer_ids?.map((id: number) => usersMap?.[id])
          const transformedDefaultReviewersArray = transformDataToArray(defaultReviewersUsers || [])
          const defaultReviewersList = transformedDefaultReviewersArray?.map(
            user => `${user.id} ${user.display_name} (${user.email})`
          )

          const minReviewerCheck = (pullreq?.approvals?.require_minimum_count as number) > 0
          const minDefaultReviewerCheck = (pullreq?.approvals?.require_minimum_default_reviewer_count as number) > 0
          const isMergePresent = pullreq?.merge?.strategies_allowed?.includes(MergeStrategy.MERGE)
          const isSquashPresent = pullreq?.merge?.strategies_allowed?.includes(MergeStrategy.SQUASH)
          const isRebasePresent = pullreq?.merge?.strategies_allowed?.includes(MergeStrategy.REBASE)
          const isFFMergePresent = pullreq?.merge?.strategies_allowed?.includes(MergeStrategy.FAST_FORWARD)

          return {
            ...commonRulesForm,
            defaultReviewersEnabled: (pullreq?.reviewers?.default_reviewer_ids?.length || 0) > 0,
            defaultReviewersList: defaultReviewersList,
            requireMinReviewers: minReviewerCheck,
            requireMinDefaultReviewers: minDefaultReviewerCheck,
            minReviewers: minReviewerCheck ? pullreq?.approvals?.require_minimum_count : '',
            minDefaultReviewers: minDefaultReviewerCheck
              ? pullreq?.approvals?.require_minimum_default_reviewer_count
              : '',
            autoAddCodeOwner: pullreq?.reviewers?.request_code_owners,
            requireCodeOwner: pullreq?.approvals?.require_code_owners,
            requireNewChanges: pullreq?.approvals?.require_latest_commit,
            reqResOfChanges: pullreq?.approvals?.require_no_change_request,
            requireCommentResolution: pullreq?.comments?.require_resolve_all,
            requireStatusChecks: (pullreq?.status_checks?.require_identifiers?.length || 0) > 0,
            statusChecks: pullreq?.status_checks?.require_identifiers || [],
            limitMergeStrategies: !!pullreq?.merge?.strategies_allowed,
            mergeCommit: isMergePresent,
            squashMerge: isSquashPresent,
            rebaseMerge: isRebasePresent,
            fastForwardMerge: isFFMergePresent,
            autoDelete: pullreq?.merge?.delete_branch,
            blockCreation: lifecycle?.create_forbidden,
            blockUpdate: lifecycle?.update_forbidden && pullreq?.merge?.block,
            blockDeletion: lifecycle?.delete_forbidden,
            blockForcePush: lifecycle?.update_forbidden || lifecycle?.update_force_forbidden,
            requirePr: lifecycle?.update_forbidden && !pullreq?.merge?.block,
            defaultReviewersSet: false
          }
        }
        case ProtectionRulesType.TAG: {
          const { lifecycle } = (definition as ProtectionTag) || {}
          return {
            ...commonRulesForm,
            blockCreation: lifecycle?.create_forbidden,
            blockUpdate: lifecycle?.update_force_forbidden,
            blockDeletion: lifecycle?.delete_forbidden
          } as RulesFormPayload
        }
      }
    }

    return rulesFormInitialPayload // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editMode, rule, currentRuleScope, principals])

  const permPushResult = hooks?.usePermissionTranslate(
    getEditPermissionRequestFromScope(space, currentRuleScope ?? 0, repoMetadata),
    [space, currentRuleScope, repoMetadata]
  )

  return (
    <Formik<RulesFormPayload>
      formName="branchProtectionRulesNewEditForm"
      initialValues={initialValues}
      enableReinitialize
      validationSchema={yup.object().shape({
        name: yup.string().trim().required().matches(REGEX_VALID_REPO_NAME, getString('validation.nameLogic')),
        minReviewers: yup.number().typeError(getString('enterANumber')),
        minDefaultReviewers: yup.number().typeError(getString('enterANumber')),
        defaultReviewersList: yup
          .array()
          .of(yup.string())
          .test(
            'min-reviewers', // Name of the test
            getString('protectionRules.atLeastMinReviewer', { count: 1 }),
            function (defaultReviewersList) {
              const { minDefaultReviewers, requireMinDefaultReviewers, defaultReviewersEnabled } = this.parent
              const minReviewers = Number(minDefaultReviewers) || 0
              if (defaultReviewersEnabled && requireMinDefaultReviewers) {
                const isValid = defaultReviewersList && defaultReviewersList.length >= minReviewers

                return (
                  isValid ||
                  this.createError({
                    message:
                      minReviewers > 1
                        ? getString('protectionRules.atLeastMinReviewers', { count: minReviewers })
                        : getString('protectionRules.atLeastMinReviewer', { count: minReviewers })
                  })
                )
              }

              return true
            }
          )
      })}
      onSubmit={async (formData, { resetForm }) => {
        const payload = getPayload(formData, ruleType)

        if (editMode) {
          handleSubmit(
            updateRule(payload),
            getString('protectionRules.ruleUpdated', { ruleType: capitalize(ruleType) }),
            resetForm
          )
        } else {
          handleSubmit(
            mutate(payload),
            getString('protectionRules.ruleCreated', { ruleType: capitalize(ruleType) }),
            resetForm
          )
        }
      }}>
      {formik => {
        const targetList =
          settingSectionMode === SettingTypeMode.EDIT || formik.values.targetSet ? formik.values.targetList : []
        const bypassList =
          settingSectionMode === SettingTypeMode.EDIT || formik.values.bypassSet ? formik.values.bypassList : []

        const bypassListOptions = getBypassListOptions(bypassList)

        const renderPrincipalIcon = (type: PrincipalType, displayName: string) => {
          switch (type) {
            case PrincipalType.USER_GROUP:
              return <Icon name="user-groups" className={cx(css.avatar, css.icon, css.ugicon)} size={24} />
            case PrincipalType.SERVICE_ACCOUNT:
              return <Icon name="service-accounts" className={cx(css.avatar, css.icon, css.saicon)} size={24} />
            case PrincipalType.USER:
            default:
              return <Avatar className={css.avatar} name={displayName} size="normal" hoverCard={false} />
          }
        }

        return (
          <FormikForm>
            <Layout.Vertical spacing={'medium'} className={css.main} padding="xlarge">
              <Container className={css.generalContainer}>
                <Layout.Horizontal flex={{ align: 'center-center' }}>
                  <Text
                    className={css.headingSize}
                    padding={{ bottom: 'medium' }}
                    font={{ variation: FontVariation.H4 }}>
                    {editMode
                      ? getString('protectionRules.edit', {
                          ruleType: capitalize(ruleType)
                        })
                      : getString('protectionRules.create', {
                          ruleType: capitalize(ruleType)
                        })}
                  </Text>
                  <FormInput.CheckBox
                    margin={{ top: 'medium', left: 'medium' }}
                    label={getString('Enable')}
                    name="enable"
                  />
                  <FlexExpander />
                </Layout.Horizontal>
                <FormInput.Text
                  name="name"
                  label={getString('name')}
                  placeholder={getString('protectionRules.namePlaceholder')}
                  tooltipProps={{
                    dataTooltipId: 'branchProtectionName'
                  }}
                  disabled={editMode}
                  className={cx(css.widthContainer, css.label)}
                />
                <FormInput.TextArea
                  name="desc"
                  label={getString('description')}
                  placeholder={getString('protectionRules.descPlaceholder')}
                  tooltipProps={{
                    dataTooltipId: 'branchProtectionDesc'
                  }}
                  className={cx(css.widthContainer, css.label)}
                />
              </Container>

              <Container className={css.generalContainer}>
                <Layout.Horizontal>
                  <FormInput.Text
                    name="target"
                    label={
                      <Layout.Vertical className={cx(css.checkContainer, css.targetContainer)}>
                        <Text
                          className={css.headingSize}
                          padding={{ bottom: 'small' }}
                          font={{ variation: FontVariation.H4 }}>
                          {getString('protectionRules.targetBranches')}
                        </Text>
                        {ruleType === ProtectionRulesType.BRANCH && (
                          <Container padding={{ top: 'small' }}>
                            <FormInput.CheckBox
                              label={getString('protectionRules.defaultBranch')}
                              name={'targetDefault'}
                            />
                          </Container>
                        )}
                      </Layout.Vertical>
                    }
                    placeholder={getString('protectionRules.targetPlaceholder')}
                    tooltipProps={{
                      dataTooltipId: 'branchProtectionTarget'
                    }}
                    className={cx(css.widthContainer, css.targetSpacingContainer, css.label)}
                  />
                  <Container
                    flex={{ alignItems: 'flex-end' }}
                    padding={{ left: 'medium' }}
                    className={css.targetSpacingContainer}>
                    <SplitButton
                      className={css.buttonContainer}
                      variation={ButtonVariation.TERTIARY}
                      text={
                        <Container flex={{ alignItems: 'center' }}>
                          <img
                            width={16}
                            height={16}
                            src={targetType === RulesTargetType.INCLUDE ? Include : Exclude}
                          />
                          <Text
                            padding={{ left: 'xsmall' }}
                            color={Color.BLACK}
                            font={{ variation: FontVariation.BODY2_SEMI, weight: 'bold' }}>
                            {getString(targetType)}
                          </Text>
                        </Container>
                      }
                      popoverProps={{
                        interactionKind: 'click',
                        usePortal: true,
                        popoverClassName: css.popover,
                        position: PopoverPosition.BOTTOM_RIGHT
                      }}
                      onClick={() => {
                        if (formik.values.target !== '') {
                          formik.setFieldValue('targetSet', true)
                          targetList.push([targetType, formik.values.target ?? ''])
                          formik.setFieldValue('targetList', targetList)
                          formik.setFieldValue('target', '')
                        }
                      }}>
                      {Object.values(RulesTargetType).map(type => (
                        <Menu.Item
                          key={type}
                          className={css.menuItem}
                          text={
                            <Container flex={{ justifyContent: 'flex-start' }}>
                              <Icon name={type === targetType ? CodeIcon.Tick : CodeIcon.Blank} />
                              <Text
                                padding={{ left: 'xsmall' }}
                                color={Color.BLACK}
                                font={{ variation: FontVariation.BODY2_SEMI, weight: 'bold' }}>
                                {getString(type)}
                              </Text>
                            </Container>
                          }
                          onClick={() => setTargetType(type)}
                        />
                      ))}
                    </SplitButton>
                  </Container>
                </Layout.Horizontal>
                <Text className={css.hintText} margin={{ bottom: 'medium' }}>
                  {getString('protectionRules.targetPatternHint')}
                </Text>
                {!isEmpty(targetList) && (
                  <Layout.Horizontal spacing={'small'} className={css.targetBox}>
                    {targetList.map((target, idx) => {
                      return (
                        <Container key={`${idx}-${target[1]}`} className={css.greyButton}>
                          <img width={16} height={16} src={target[0] === RulesTargetType.INCLUDE ? Include : Exclude} />
                          <Text lineClamp={1}>{target[1]}</Text>
                          <Icon
                            name="code-close"
                            onClick={() => {
                              const filteredData = targetList.filter(
                                item => !(item[0] === target[0] && item[1] === target[1])
                              )
                              formik.setFieldValue('targetList', filteredData)
                            }}
                            className={css.codeClose}
                          />
                        </Container>
                      )
                    })}
                  </Layout.Horizontal>
                )}
              </Container>

              <Container className={css.generalContainer}>
                <Text className={css.headingSize} padding={{ bottom: 'medium' }} font={{ variation: FontVariation.H4 }}>
                  {getString('protectionRules.bypassList')}
                </Text>
                <FormInput.CheckBox label={getString('protectionRules.allRepoOwners')} name={'allRepoOwners'} />
                <SearchDropDown
                  searchTerm={searchTerm}
                  placeholder={standalone ? getString('selectUsers') : getString('selectUsersUserGroupsAndServiceAcc')}
                  className={css.widthContainer}
                  onChange={setSearchTerm}
                  options={bypassListOptions}
                  loading={loadingPrincipals || loadingUsersGroups}
                  itemRenderer={(item, { handleClick, isActive }) => {
                    const { id, type, display_name, email_or_identifier } = JSON.parse(item.value.toString())
                    return (
                      <Layout.Horizontal
                        key={id}
                        onClick={handleClick}
                        padding={{ top: 'xsmall', bottom: 'xsmall' }}
                        flex={{ align: 'center-center' }}
                        className={cx({ [css.activeMenuItem]: isActive })}>
                        {renderPrincipalIcon(type as PrincipalType, display_name)}
                        <Layout.Vertical padding={{ left: 'small' }} width={400}>
                          <Text className={css.truncateText}>
                            <strong>{display_name}</strong>
                          </Text>
                          <Text className={css.truncateText} lineClamp={1}>
                            {email_or_identifier}
                          </Text>
                        </Layout.Vertical>
                      </Layout.Horizontal>
                    )
                  }}
                  onClick={menuItem => {
                    const value = JSON.parse(menuItem.value.toString()) as NormalizedPrincipal
                    const updatedList = [value, ...(bypassList || [])]
                    const uniqueArr = Array.from(new Map(updatedList.map(item => [item.id, item])).values())
                    formik.setFieldValue('bypassList', uniqueArr)
                    formik.setFieldValue('bypassSet', true)
                  }}
                />
                <BypassList
                  renderPrincipalIcon={renderPrincipalIcon}
                  bypassList={bypassList}
                  setFieldValue={formik.setFieldValue}
                />
              </Container>

              <Container margin={{ top: 'medium' }} className={css.generalContainer}>
                <Text className={css.headingSize} padding={{ bottom: 'medium' }} font={{ variation: FontVariation.H4 }}>
                  {getString('protectionRules.protectionSelectAll')}
                </Text>
                {ruleType === ProtectionRulesType.BRANCH ? (
                  <BranchRulesForm
                    formik={formik}
                    statusOptions={statusOptions}
                    setSearchStatusTerm={setSearchStatusTerm}
                    defaultReviewerProps={{
                      setSearchTerm,
                      userPrincipalOptions
                    }}
                  />
                ) : (
                  <TagRulesForm />
                )}
              </Container>

              <Container padding={{ top: 'large' }}>
                <Layout.Horizontal spacing="small">
                  <Button
                    onClick={() => {
                      formik.submitForm()
                    }}
                    type="button"
                    text={editMode ? getString('protectionRules.saveRule') : getString('protectionRules.createRule')}
                    variation={ButtonVariation.PRIMARY}
                    disabled={false}
                    {...permissionProps(permPushResult, standalone)}
                  />
                  <Button
                    text={getString('cancel')}
                    variation={ButtonVariation.TERTIARY}
                    onClick={() => {
                      history.goBack()
                    }}
                  />
                </Layout.Horizontal>
              </Container>
            </Layout.Vertical>
          </FormikForm>
        )
      }}
    </Formik>
  )
}

export default ProtectionRulesForm
