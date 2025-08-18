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

import React from 'react'
import cx from 'classnames'
import { Container, FlexExpander, FormInput, Layout, SelectOption, Text } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import type { FormikProps } from 'formik'
import { Classes, Popover, PopoverInteractionKind, PopoverPosition } from '@blueprintjs/core'
import { useStrings } from 'framework/strings'
import { ProtectionRulesType } from 'utils/GitUtils'
import type { RulesFormPayload } from 'components/ProtectionRules/ProtectionRulesUtils'
import DefaultReviewersSection from './DefaultReviewersSection'
import css from '../ProtectionRulesForm.module.scss'

const BranchRulesForm = (props: {
  statusOptions: SelectOption[]
  setSearchStatusTerm: React.Dispatch<React.SetStateAction<string>>
  formik: FormikProps<RulesFormPayload>
  defaultReviewerProps: {
    setSearchTerm: React.Dispatch<React.SetStateAction<string>>
    userPrincipalOptions: SelectOption[]
  }
}) => {
  const { formik, defaultReviewerProps, setSearchStatusTerm, statusOptions } = props
  const { getString } = useStrings()
  const { setFieldValue } = formik
  const {
    blockUpdate,
    blockForcePush,
    limitMergeStrategies,
    requireMinReviewers,
    requirePr,
    requireStatusChecks,
    statusChecks
  } = formik.values

  const filteredStatusOptions = statusOptions.filter(
    (item: SelectOption) => !statusChecks?.includes(item.value as string)
  )

  return (
    <>
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.blockCreation', { ruleType: ProtectionRulesType.BRANCH })}
        name={'blockCreation'}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.blockCreationText', { refs: 'branches' })}
      </Text>

      <hr className={css.dividerContainer} />
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.blockDeletion', { refs: 'branches' })}
        name={'blockDeletion'}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.blockDeletionText', { refs: 'branches' })}
      </Text>

      <hr className={css.dividerContainer} />
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.blockUpdate', { refs: 'branches' })}
        name={'blockUpdate'}
        onChange={() => {
          setFieldValue('blockForcePush', !(blockUpdate && blockForcePush))
          setFieldValue('requirePr', false)
        }}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.blockUpdateText', { refs: 'branches' })}
      </Text>

      <hr className={css.dividerContainer} />
      <Popover
        interactionKind={PopoverInteractionKind.HOVER}
        position={PopoverPosition.TOP_LEFT}
        popoverClassName={Classes.DARK}
        disabled={!(blockUpdate || requirePr)}
        content={
          <Container padding="medium">
            <Text font={{ variation: FontVariation.FORM_HELP }} color={Color.WHITE}>
              {requirePr ? getString('pushBlockedMessage') : getString('ruleBlockedMessage')}
            </Text>
          </Container>
        }>
        <>
          <FormInput.CheckBox
            disabled={blockUpdate || requirePr}
            className={css.checkboxLabel}
            label={getString('protectionRules.blockForcePush')}
            name={'blockForcePush'}
          />
          <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
            {getString('protectionRules.blockForcePushText')}
          </Text>
        </>
      </Popover>

      <hr className={css.dividerContainer} />
      <Popover
        interactionKind={PopoverInteractionKind.HOVER}
        position={PopoverPosition.TOP_LEFT}
        popoverClassName={Classes.DARK}
        disabled={!blockUpdate}
        content={
          <Container padding="medium">
            <Text font={{ variation: FontVariation.FORM_HELP }} color={Color.WHITE}>
              {getString('ruleBlockedMessage')}
            </Text>
          </Container>
        }>
        <>
          <FormInput.CheckBox
            disabled={blockUpdate}
            className={css.checkboxLabel}
            label={getString('protectionRules.requirePr')}
            name={'requirePr'}
            onChange={() => {
              setFieldValue('blockForcePush', !requirePr)
            }}
          />
          <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
            {getString('protectionRules.requirePrText')}
          </Text>
        </>
      </Popover>

      <hr className={css.dividerContainer} />
      <DefaultReviewersSection formik={formik} defaultReviewerProps={defaultReviewerProps} />

      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.requireMinReviewersTitle')}
        name={'requireMinReviewers'}
        onChange={e => {
          if ((e.target as HTMLInputElement).checked) {
            setFieldValue('minReviewers', 1)
          }
        }}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.requireMinReviewersContent')}
      </Text>
      {requireMinReviewers && (
        <Container padding={{ left: 'xlarge', top: 'medium' }}>
          <FormInput.Text
            inputGroup={{ type: 'number' }}
            className={cx(css.widthContainer, css.minText)}
            name={'minReviewers'}
            placeholder={getString('protectionRules.minNumberPlaceholder')}
            label={getString('protectionRules.minNumber')}
          />
        </Container>
      )}

      <hr className={css.dividerContainer} />
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.addCodeownersToReviewTitle')}
        name={'autoAddCodeOwner'}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.addCodeownersToReviewText')}
      </Text>

      <hr className={css.dividerContainer} />
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.reqReviewFromCodeOwnerTitle')}
        name={'requireCodeOwner'}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.reqReviewFromCodeOwnerText')}
      </Text>

      <hr className={css.dividerContainer} />
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.reqNewChangesTitle')}
        name={'requireNewChanges'}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.reqNewChangesText')}
      </Text>

      <hr className={css.dividerContainer} />
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.reqResOfChanges')}
        name={'reqResOfChanges'}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.reqResOfChangesText')}
      </Text>

      <hr className={css.dividerContainer} />
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.reqCommentResolutionTitle')}
        name={'requireCommentResolution'}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.reqCommentResolutionText')}
      </Text>

      <hr className={css.dividerContainer} />
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.reqStatusChecksTitle')}
        name={'requireStatusChecks'}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.reqStatusChecksText')}
      </Text>
      {requireStatusChecks && (
        <Container padding={{ left: 'xlarge', top: 'large' }}>
          <FormInput.Select
            className={css.widthContainer}
            onQueryChange={setSearchStatusTerm}
            items={filteredStatusOptions}
            value={{ label: '', value: '' }}
            placeholder={getString('selectStatuses')}
            onChange={item => {
              statusChecks?.push(item.value as string)
              const uniqueArr = Array.from(new Set(statusChecks))
              setFieldValue('statusChecks', uniqueArr)
            }}
            label={getString('protectionRules.statusCheck')}
            name={'statusSelect'}
          />
          <Container className={css.bypassContainer}>
            {statusChecks?.map((status, idx) => (
              <Layout.Horizontal key={`${idx}-${status[1]}`} flex={{ align: 'center-center' }} padding={'small'}>
                <Text padding={{ top: 'tiny' }} lineClamp={1}>
                  {status}
                </Text>
                <FlexExpander />
                <Icon
                  className={css.codeClose}
                  name="code-close"
                  onClick={() => {
                    const filteredData = statusChecks.filter(item => !(item === status))
                    setFieldValue('statusChecks', filteredData)
                  }}
                />
              </Layout.Horizontal>
            ))}
          </Container>
        </Container>
      )}

      <hr className={css.dividerContainer} />
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.limitMergeStrategies')}
        name={'limitMergeStrategies'}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.limitMergeStrategiesText')}
      </Text>
      {limitMergeStrategies && (
        <Container padding={{ left: 'xlarge', top: 'large' }}>
          <Container
            padding={{ top: 'medium', left: 'medium', right: 'medium', bottom: 'small' }}
            className={cx(css.widthContainer, css.greyContainer)}>
            {['mergeCommit', 'squashMerge', 'rebaseMerge', 'fastForwardMerge'].map(method => (
              <FormInput.CheckBox key={method} className={css.minText} label={getString(method as any)} name={method} />
            ))}
          </Container>
        </Container>
      )}

      <hr className={css.dividerContainer} />
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.autoDeleteTitle')}
        name={'autoDelete'}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.autoDeleteText')}
      </Text>
    </>
  )
}

export default BranchRulesForm
