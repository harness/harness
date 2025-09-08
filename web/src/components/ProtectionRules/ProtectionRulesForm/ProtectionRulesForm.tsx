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
import cx from 'classnames'
import * as yup from 'yup'
import {
  Button,
  ButtonVariation,
  Container,
  FlexExpander,
  FormInput,
  Formik,
  FormikForm,
  Layout,
  Text,
  useToaster
} from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useHistory, useParams } from 'react-router-dom'
import { useGet, useMutate } from 'restful-react'
import { capitalize, isEmpty } from 'lodash-es'
import { useStrings } from 'framework/strings'
import { MergeStrategy, ProtectionRulesType, RulesTargetType, SettingsTab } from 'utils/GitUtils'
import {
  ScopeEnum,
  REGEX_VALID_REPO_NAME,
  getEditPermissionRequestFromScope,
  getErrorMessage,
  getScopeData,
  getScopeFromParams,
  permissionProps,
  PrincipalType,
  combineAndNormalizePrincipalsAndGroups,
  NormalizedPrincipal
} from 'utils/Utils'
import type {
  RepoRepositoryOutput,
  OpenapiRule,
  TypesPrincipalInfo,
  ProtectionBranch,
  ProtectionTag,
  TypesUserGroupInfo
} from 'services/code'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { getConfig } from 'services/config'
import type { Identifier } from 'utils/types'
import SearchDropDown, { renderPrincipalIcon } from 'components/SearchDropDown/SearchDropDown'
import { useQueryParams } from 'hooks/useQueryParams'
import BranchRulesForm from './components/BranchRulesForm'
import TagRulesForm from './components/TagRulesForm'
import {
  convertToTargetList,
  getFilteredNormalizedPrincipalOptions,
  getPayload,
  rulesFormInitialPayload,
  RulesFormPayload,
  RuleState,
  transformDataToArray
} from '../ProtectionRulesUtils'
import TargetPatternsSection from './components/TargetPatternsSection'
import TargetRepositoriesSection from './components/TargetRepositoriesSection'
import NormalizedPrincipalsList from './components/NormalizedPrincipalsList'
import css from './ProtectionRulesForm.module.scss'

const ProtectionRulesForm = (props: {
  currentPageScope: ScopeEnum
  editMode: boolean
  repoMetadata?: RepoRepositoryOutput
}) => {
  const { routes, routingId, standalone, hooks } = useAppContext()
  const params = useParams<Identifier>()
  const queryParams = useQueryParams<{ type: ProtectionRulesType }>()
  const { ruleId } = useGetRepositoryMetadata()
  const { showError, showSuccess } = useToaster()
  const space = useGetSpaceParam()
  const history = useHistory()
  const { currentPageScope, editMode = false, repoMetadata } = props
  const { getString } = useStrings()
  const [searchTerm, setSearchTerm] = useState('')
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

  const { data: userGroups, loading: loadingUsersGroups } = useGet<TypesUserGroupInfo[]>({
    path: `/api/v1/usergroups`,
    queryParams: {
      query: searchTerm,
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
        repo_target,
        state,
        users: usersMap,
        user_groups: userGroupsMap,
        repositories: repositoriesMap
      } = rule || {}
      const { bypass } = definition || {}

      const bypassListUsers = bypass?.user_ids?.map((id: number) => usersMap?.[id])
      const bypassListUserGroups = bypass?.user_group_ids?.map(id => userGroupsMap?.[id])
      const transformedBypassList = transformDataToArray(bypassListUsers || [])
      const transformedUserGroupsBypassList = transformDataToArray(bypassListUserGroups || [])
      const bypassList = combineAndNormalizePrincipalsAndGroups(transformedBypassList, transformedUserGroupsBypassList)

      // Create a new array based on the "include" key from the JSON object and the strings array
      const includeArr = convertToTargetList(pattern?.include, true)
      const excludeArr = convertToTargetList(pattern?.exclude, false)
      const repoIncludeArr =
        repo_target?.include?.ids?.map(id => [
          RulesTargetType.INCLUDE,
          repositoriesMap?.[id]?.path || '',
          String(id)
        ]) ?? []
      const repoExcludeArr =
        repo_target?.exclude?.ids?.map(id => [
          RulesTargetType.EXCLUDE,
          repositoriesMap?.[id]?.path || '',
          String(id)
        ]) ?? []
      const repoPatternsIncludeArr = convertToTargetList(repo_target?.include?.patterns, true)
      const repoPatternsExcludeArr = convertToTargetList(repo_target?.exclude?.patterns, false)

      const commonRulesForm = {
        name: identifier,
        desc: description,
        enable: state !== RuleState.DISABLED,
        repoList: [...repoIncludeArr, ...repoExcludeArr],
        repoTargetList: [...repoPatternsIncludeArr, ...repoPatternsExcludeArr],
        targetList: [...includeArr, ...excludeArr],
        allRepoOwners: bypass?.repo_owners,
        bypassList: bypassList
      }

      switch (ruleType) {
        case ProtectionRulesType.BRANCH: {
          const { lifecycle, pullreq } = (definition as ProtectionBranch) || {}

          const defaultReviewerUsers = pullreq?.reviewers?.default_reviewer_ids?.map((id: number) => usersMap?.[id])
          const defaultReviewerUserGroups = pullreq?.reviewers?.default_user_group_reviewer_ids?.map(
            (id: number) => userGroupsMap?.[id]
          )
          const transformedDefaultReviewers = transformDataToArray(defaultReviewerUsers || [])
          const transformedDefaultUserGroupReviewers = transformDataToArray(defaultReviewerUserGroups || [])
          const defaultReviewersList = combineAndNormalizePrincipalsAndGroups(
            transformedDefaultReviewers,
            transformedDefaultUserGroupReviewers
          )

          const minReviewerCheck = (pullreq?.approvals?.require_minimum_count as number) > 0
          const minDefaultReviewerCheck = (pullreq?.approvals?.require_minimum_default_reviewer_count as number) > 0
          const isMergePresent = pullreq?.merge?.strategies_allowed?.includes(MergeStrategy.MERGE)
          const isSquashPresent = pullreq?.merge?.strategies_allowed?.includes(MergeStrategy.SQUASH)
          const isRebasePresent = pullreq?.merge?.strategies_allowed?.includes(MergeStrategy.REBASE)
          const isFFMergePresent = pullreq?.merge?.strategies_allowed?.includes(MergeStrategy.FAST_FORWARD)

          return {
            ...commonRulesForm,
            targetDefaultBranch: pattern?.default,
            defaultReviewersEnabled: !isEmpty(defaultReviewersList),
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
            requirePr: lifecycle?.update_forbidden && !pullreq?.merge?.block
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
        const { bypassList = [] } = formik.values

        const bypassListOptions = getFilteredNormalizedPrincipalOptions(combinedOptions, bypassList)

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

              {currentRuleScope !== ScopeEnum.REPO_SCOPE && (
                <Container className={css.generalContainer}>
                  <Text
                    className={css.headingSize}
                    padding={{ bottom: 'medium' }}
                    font={{ variation: FontVariation.H4 }}>
                    {getString('protectionRules.targetRepositories')}
                  </Text>
                  {/* <FormInput.Select
                  className={css.widthContainer}
                  label={getString('scope')}
                  name="repoScope"
                  items={getScopeOptions(getString, accountIdentifier, orgIdentifier)}
                /> */}
                  <TargetRepositoriesSection
                    formik={formik}
                    space={space}
                    standalone={standalone}
                    currentScope={currentRuleScope}
                  />
                  <Text
                    font={{ variation: FontVariation.FORM_LABEL }}
                    className={css.labelText}
                    padding={{ bottom: 'xsmall' }}>
                    {getString('patterns')}
                  </Text>
                  <TargetPatternsSection formik={formik} repoTarget ruleType={ruleType} />
                </Container>
              )}

              <Container className={css.generalContainer}>
                <Layout.Vertical className={cx(css.checkContainer, css.targetContainer)}>
                  <Text
                    className={css.headingSize}
                    padding={{ bottom: 'small' }}
                    font={{ variation: FontVariation.H4 }}>
                    {getString('protectionRules.targetPatterns')}
                  </Text>
                  {ruleType === ProtectionRulesType.BRANCH && (
                    <Container padding={{ top: 'small', bottom: 'medium' }}>
                      <FormInput.CheckBox
                        label={getString('protectionRules.defaultBranch')}
                        name={'targetDefaultBranch'}
                      />
                    </Container>
                  )}
                </Layout.Vertical>
                <TargetPatternsSection formik={formik} tooltipId="branchProtectionTarget" ruleType={ruleType} />
              </Container>

              <Container className={css.generalContainer}>
                <Text className={css.headingSize} padding={{ bottom: 'medium' }} font={{ variation: FontVariation.H4 }}>
                  {getString('protectionRules.bypassList')}
                </Text>
                <FormInput.CheckBox label={getString('protectionRules.allRepoOwners')} name={'allRepoOwners'} />
                <SearchDropDown
                  searchTerm={searchTerm}
                  placeholder={getString('selectUsersUserGroupsAndServiceAcc')}
                  className={css.widthContainer}
                  onChange={setSearchTerm}
                  options={bypassListOptions}
                  loading={loadingPrincipals || loadingUsersGroups}
                  itemRenderer={(item, { handleClick, isActive }) => {
                    const { id, type, display_name, email_or_identifier } = JSON.parse(item.value.toString())
                    return (
                      <Layout.Horizontal
                        key={`${id}-${email_or_identifier}`}
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
                  }}
                />
                <NormalizedPrincipalsList
                  fieldName={'bypassList'}
                  list={bypassList}
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
                    currentRuleScope={currentRuleScope}
                    repoMetadata={repoMetadata}
                    ruleType={ruleType}
                    scopeRef={scopeRef}
                    defaultReviewerProps={{
                      loading: loadingPrincipals || loadingUsersGroups,
                      searchTerm,
                      setSearchTerm,
                      normalizedPrincipalOptions: combinedOptions
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
