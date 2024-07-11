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
  SelectOption,
  SplitButton,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { Menu, PopoverPosition } from '@blueprintjs/core'
import { Icon } from '@harnessio/icons'
import { useHistory } from 'react-router-dom'
import { useGet, useMutate } from 'restful-react'
import { BranchTargetType, SettingTypeMode, SettingsTab, branchTargetOptions } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { REGEX_VALID_REPO_NAME, getErrorMessage, permissionProps, rulesFormInitialPayload } from 'utils/Utils'
import type {
  RepoRepositoryOutput,
  OpenapiRule,
  TypesPrincipalInfo,
  EnumMergeMethod,
  ProtectionPattern,
  ProtectionBranch
} from 'services/code'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import ProtectionRulesForm from './ProtectionRulesForm/ProtectionRulesForm'
import Include from '../../../icons/Include.svg?url'
import Exclude from '../../../icons/Exclude.svg?url'
import BypassList from './BypassList'
import css from './BranchProtectionForm.module.scss'

const BranchProtectionForm = (props: {
  ruleUid: string
  editMode: boolean
  repoMetadata?: RepoRepositoryOutput | undefined
  refetchRules: () => void
  settingSectionMode: SettingTypeMode
}) => {
  const { routes, routingId, standalone, hooks } = useAppContext()

  const { ruleId } = useGetRepositoryMetadata()
  const { showError, showSuccess } = useToaster()
  const { editMode = false, repoMetadata, ruleUid, refetchRules, settingSectionMode } = props
  const { getString } = useStrings()
  const { data: rule } = useGet<OpenapiRule>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/rules/${ruleId}`,
    lazy: !repoMetadata && !ruleId
  })
  const [searchTerm, setSearchTerm] = useState('')
  const [searchStatusTerm, setSearchStatusTerm] = useState('')

  const { mutate } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata?.path}/+/rules/`
  })

  const { mutate: updateRule } = useMutate({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoMetadata?.path}/+/rules/${ruleId}`
  })
  const { data: users } = useGet<TypesPrincipalInfo[]>({
    path: `/api/v1/principals`,
    queryParams: {
      query: searchTerm,
      type: 'user',
      accountIdentifier: routingId,
      debounce: 500
    }
  })
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const transformDataToArray = (data: any) => {
    return Object.keys(data).map(key => {
      return {
        ...data[key]
      }
    })
  }
  const transformUserArray = transformDataToArray(rule?.users || [])
  const usersArrayCurr = transformUserArray?.map(user => `${user.id} ${user.display_name}`)
  const [userArrayState, setUserArrayState] = useState<string[]>(usersArrayCurr)

  const { data: statuses } = useGet<string[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/checks/recent`,
    queryParams: {
      query: searchStatusTerm,
      debounce: 500
    }
  })
  const statusOptions: SelectOption[] = useMemo(
    () =>
      statuses?.map(status => ({
        value: status,
        label: status
      })) || [],
    [statuses]
  )
  const userOptions: SelectOption[] = useMemo(
    () =>
      users?.map(user => ({
        value: `${user.id?.toString() as string} ${user.uid}`,
        label: (user.display_name || user.email) as string
      })) || [],
    [users]
  )

  const handleSubmit = async (operation: Promise<OpenapiRule>, successMessage: string, resetForm: () => void) => {
    try {
      await operation
      showSuccess(successMessage)
      resetForm()
      history.push(
        routes.toCODESettings({
          repoPath: repoMetadata?.path as string,
          settingSection: SettingsTab.branchProtection
        })
      )
      refetchRules?.()
    } catch (exception) {
      showError(getErrorMessage(exception))
    }
  }
  const history = useHistory()

  const initialValues = useMemo(() => {
    if (editMode && rule) {
      const minReviewerCheck =
        ((rule.definition as ProtectionBranch)?.pullreq?.approvals?.require_minimum_count as number) > 0 ? true : false
      const isMergePresent = (rule.definition as ProtectionBranch)?.pullreq?.merge?.strategies_allowed?.includes(
        'merge'
      )
      const isSquashPresent = (rule.definition as ProtectionBranch)?.pullreq?.merge?.strategies_allowed?.includes(
        'squash'
      )
      const isRebasePresent = (rule.definition as ProtectionBranch)?.pullreq?.merge?.strategies_allowed?.includes(
        'rebase'
      )
      // List of strings to be included in the final array
      const includeList = (rule?.pattern as ProtectionPattern)?.include ?? []
      const excludeList = (rule?.pattern as ProtectionPattern)?.exclude ?? []
      // Create a new array based on the "include" key from the JSON object and the strings array
      const includeArr = includeList?.map((arr: string) => ['include', arr])
      const excludeArr = excludeList?.map((arr: string) => ['exclude', arr])
      const finalArray = [...includeArr, ...excludeArr]
      const usersArray = transformDataToArray(rule.users)
      const bypassList =
        userArrayState.length > 0 ? userArrayState : usersArray?.map(user => `${user.id} ${user.display_name}`)

      return {
        name: rule?.identifier,
        desc: rule.description,
        enable: rule.state !== 'disabled',
        target: '',
        targetDefault: (rule?.pattern as ProtectionPattern)?.default,
        targetList: finalArray,
        allRepoOwners: (rule.definition as ProtectionBranch)?.bypass?.repo_owners,
        bypassList: bypassList,
        requireMinReviewers: minReviewerCheck,
        minReviewers: minReviewerCheck
          ? (rule.definition as ProtectionBranch)?.pullreq?.approvals?.require_minimum_count
          : '',
        requireCodeOwner: (rule.definition as ProtectionBranch)?.pullreq?.approvals?.require_code_owners,
        requireNewChanges: (rule.definition as ProtectionBranch)?.pullreq?.approvals?.require_latest_commit,
        reqResOfChanges: (rule.definition as ProtectionBranch)?.pullreq?.approvals?.require_no_change_request,
        requireCommentResolution: (rule.definition as ProtectionBranch)?.pullreq?.comments?.require_resolve_all, // eslint-disable-next-line @typescript-eslint/no-explicit-any
        requireStatusChecks: (rule.definition as any)?.pullreq?.status_checks?.require_identifiers?.length > 0,
        statusChecks:
          (rule.definition as ProtectionBranch)?.pullreq?.status_checks?.require_identifiers || ([] as string[]),
        limitMergeStrategies: !!(rule.definition as ProtectionBranch)?.pullreq?.merge?.strategies_allowed,
        mergeCommit: isMergePresent,
        squashMerge: isSquashPresent,
        rebaseMerge: isRebasePresent,
        autoDelete: (rule.definition as ProtectionBranch)?.pullreq?.merge?.delete_branch,
        blockBranchCreation: (rule.definition as ProtectionBranch)?.lifecycle?.create_forbidden,
        blockBranchDeletion: (rule.definition as ProtectionBranch)?.lifecycle?.delete_forbidden,
        requirePr: (rule.definition as ProtectionBranch)?.lifecycle?.update_forbidden,
        targetSet: false,
        bypassSet: false
      }
    }

    return rulesFormInitialPayload // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editMode, rule, ruleUid, users])
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
    <Formik
      formName="branchProtectionRulesNewEditForm"
      initialValues={initialValues}
      enableReinitialize
      validationSchema={yup.object().shape({
        name: yup.string().trim().required().matches(REGEX_VALID_REPO_NAME, getString('validation.nameLogic')),
        minReviewers: yup.number().typeError(getString('enterANumber'))
      })}
      onSubmit={async (formData, { resetForm }) => {
        const stratArray = [
          formData.squashMerge && 'squash',
          formData.rebaseMerge && 'rebase',
          formData.mergeCommit && 'merge'
        ].filter(Boolean) as EnumMergeMethod[]
        const includeArray =
          formData?.targetList?.filter(([type]) => type === 'include').map(([, value]) => value) ?? []
        const excludeArray =
          formData?.targetList?.filter(([type]) => type === 'exclude').map(([, value]) => value) ?? []

        const bypassList = formData?.bypassList?.map(item => parseInt(item.split(' ')[0]))
        const payload: OpenapiRule = {
          identifier: formData.name,
          type: 'branch',
          description: formData.desc,
          state: formData.enable === true ? 'active' : 'disabled',
          pattern: {
            default: formData.targetDefault,
            exclude: excludeArray,
            include: includeArray
          },
          definition: {
            bypass: {
              user_ids: bypassList,
              repo_owners: formData.allRepoOwners
            },
            pullreq: {
              approvals: {
                require_code_owners: formData.requireCodeOwner,
                require_minimum_count: parseInt(formData.minReviewers as string),
                require_latest_commit: formData.requireNewChanges,
                require_no_change_request: formData.reqResOfChanges
              },
              comments: {
                require_resolve_all: formData.requireCommentResolution
              },
              merge: {
                strategies_allowed: stratArray,
                delete_branch: formData.autoDelete
              },
              status_checks: {
                require_identifiers: formData.statusChecks
              }
            },
            lifecycle: {
              create_forbidden: formData.blockBranchCreation,
              delete_forbidden: formData.blockBranchDeletion,
              update_forbidden: formData.requirePr
            }
          }
        }
        if (!formData.requireStatusChecks) {
          delete (payload?.definition as ProtectionBranch)?.pullreq?.status_checks
        }
        if (!formData.limitMergeStrategies) {
          delete (payload?.definition as ProtectionBranch)?.pullreq?.merge?.strategies_allowed
        }
        if (!formData.requireMinReviewers) {
          delete (payload?.definition as ProtectionBranch)?.pullreq?.approvals?.require_minimum_count
        }
        if (editMode) {
          handleSubmit(updateRule(payload), getString('branchProtection.ruleUpdated'), resetForm)
        } else {
          handleSubmit(mutate(payload), getString('branchProtection.ruleCreated'), resetForm)
        }
      }}>
      {formik => {
        const targetList =
          settingSectionMode === SettingTypeMode.EDIT || formik.values.targetSet ? formik.values.targetList : []
        const bypassList =
          settingSectionMode === SettingTypeMode.EDIT || formik.values.bypassSet ? formik.values.bypassList : []
        const minReviewers = formik.values.requireMinReviewers
        const statusChecks = formik.values.statusChecks
        const limitMergeStrats = formik.values.limitMergeStrategies
        const requireStatusChecks = formik.values.requireStatusChecks

        const filteredUserOptions = userOptions.filter(
          (item: SelectOption) => !bypassList.includes(item.value as string)
        )

        return (
          <FormikForm>
            <Container className={css.main} padding="xlarge">
              <Container className={css.generalContainer}>
                <Layout.Horizontal flex={{ align: 'center-center' }}>
                  <Text
                    className={css.headingSize}
                    padding={{ bottom: 'medium' }}
                    font={{ variation: FontVariation.H4 }}>
                    {editMode ? getString('branchProtection.edit') : getString('branchProtection.create')}
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
                  placeholder={getString('branchProtection.namePlaceholder')}
                  tooltipProps={{
                    dataTooltipId: 'branchProtectionName'
                  }}
                  disabled={editMode}
                  className={cx(css.widthContainer, css.label)}
                />
                <FormInput.Text
                  name="desc"
                  label={getString('description')}
                  placeholder={getString('branchProtection.descPlaceholder')}
                  tooltipProps={{
                    dataTooltipId: 'branchProtectionDesc'
                  }}
                  className={cx(css.widthContainer, css.label)}
                />
                <hr className={css.dividerContainer} />
                <Layout.Horizontal>
                  <FormInput.Text
                    name="target"
                    label={
                      <Layout.Vertical className={cx(css.checkContainer, css.targetContainer)}>
                        <Text margin={{ bottom: 'small' }} className={css.label}>
                          {getString('branchProtection.targetBranches')}
                        </Text>
                        <FormInput.CheckBox
                          margin={{ bottom: 'medium' }}
                          label={getString('branchProtection.defaultBranch')}
                          name={'targetDefault'}
                        />
                      </Layout.Vertical>
                    }
                    placeholder={getString('branchProtection.targetPlaceholder')}
                    tooltipProps={{
                      dataTooltipId: 'branchProtectionTarget'
                    }}
                    className={cx(css.widthContainer, css.targetSpacingContainer, css.label)}
                  />
                  <Container
                    className={css.paddingTop}
                    flex={{ align: 'center-center' }}
                    padding={{ top: 'xxlarge', left: 'small' }}>
                    <SplitButton
                      //   className={css.buttonContainer}
                      variation={ButtonVariation.TERTIARY}
                      text={
                        <Container flex={{ alignItems: 'center' }}>
                          <img width={16} height={17} src={Include} />
                          <Text
                            padding={{ left: 'xsmall' }}
                            color={Color.BLACK}
                            font={{ variation: FontVariation.BODY2_SEMI, weight: 'bold' }}>
                            {branchTargetOptions[0].title}
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

                          targetList.push([BranchTargetType.INCLUDE, formik.values.target])
                          formik.setFieldValue('targetList', targetList)
                          formik.setFieldValue('target', '')
                        }
                      }}>
                      {[branchTargetOptions[1]].map(option => {
                        return (
                          <Menu.Item
                            className={css.menuItem}
                            key={option.type}
                            text={<Text font={{ variation: FontVariation.BODY2 }}>{option.title}</Text>}
                            onClick={() => {
                              if (formik.values.target !== '') {
                                formik.setFieldValue('targetSet', true)

                                targetList.push([BranchTargetType.EXCLUDE, formik.values.target])
                                formik.setFieldValue('targetList', targetList)
                                formik.setFieldValue('target', '')
                              }
                            }}
                          />
                        )
                      })}
                    </SplitButton>
                  </Container>
                </Layout.Horizontal>
                <Text className={css.hintText} margin={{ bottom: 'medium' }}>
                  {getString('branchProtection.targetPatternHint')}
                </Text>
                <Layout.Horizontal spacing={'small'} className={css.targetBox}>
                  {targetList.map((target, idx) => {
                    return (
                      <Container key={`${idx}-${target[1]}`} className={css.greyButton}>
                        {target[0] === BranchTargetType.INCLUDE ? (
                          <img width={16} height={17} src={Include} />
                        ) : (
                          <img width={16} height={16} src={Exclude} />
                        )}
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
              </Container>
              <Container margin={{ top: 'medium' }} className={css.generalContainer}>
                <Text className={css.headingSize} padding={{ bottom: 'medium' }} font={{ variation: FontVariation.H4 }}>
                  {getString('branchProtection.bypassList')}
                </Text>
                <FormInput.CheckBox label={getString('branchProtection.allRepoOwners')} name={'allRepoOwners'} />
                <FormInput.Select
                  items={filteredUserOptions}
                  onQueryChange={setSearchTerm}
                  className={css.widthContainer}
                  value={{ label: '', value: '' }}
                  placeholder={getString('selectUsers')}
                  onChange={item => {
                    const id = item.value?.toString().split(' ')[0]
                    const displayName = item.label
                    const bypassEntry = `${id} ${displayName}`
                    bypassList?.push(bypassEntry)
                    const uniqueArr = Array.from(new Set(bypassList))
                    formik.setFieldValue('bypassList', uniqueArr)
                    formik.setFieldValue('bypassSet', true)
                    setUserArrayState([...uniqueArr])
                  }}
                  name={'bypassSelect'}></FormInput.Select>
                <BypassList bypassList={bypassList} setFieldValue={formik.setFieldValue} />
              </Container>
              <ProtectionRulesForm
                setFieldValue={formik.setFieldValue}
                requireStatusChecks={requireStatusChecks}
                minReviewers={minReviewers}
                statusOptions={statusOptions}
                statusChecks={statusChecks}
                limitMergeStrats={limitMergeStrats}
                setSearchStatusTerm={setSearchStatusTerm}
              />
              <Container padding={{ top: 'large' }}>
                <Layout.Horizontal spacing="small">
                  <Button
                    onClick={() => {
                      formik.submitForm()
                    }}
                    type="button"
                    text={editMode ? getString('branchProtection.saveRule') : getString('branchProtection.createRule')}
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
            </Container>
          </FormikForm>
        )
      }}
    </Formik>
  )
}

export default BranchProtectionForm
