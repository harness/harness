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

import React, { useMemo } from 'react'
import cx from 'classnames'
import { Container, FormInput, SelectOption, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import type { FormikProps } from 'formik'
import { Render } from 'react-jsx-match'
import { useStrings } from 'framework/strings'
import type { RulesFormPayload } from 'utils/Utils'
import { SettingTypeMode } from 'utils/GitUtils'
import DefaultReviewersList from './DefaultReviewersList'
import css from '../ProtectionRulesForm.module.scss'

const DefaultReviewersSection = (props: {
  formik: FormikProps<RulesFormPayload>
  defaultReviewerProps: {
    setSearchTerm: React.Dispatch<React.SetStateAction<string>>
    userPrincipalOptions: SelectOption[]
    settingSectionMode: SettingTypeMode
    setDefaultReviewersState: React.Dispatch<React.SetStateAction<string[]>>
  }
}) => {
  const { formik, defaultReviewerProps } = props
  const { settingSectionMode, userPrincipalOptions, setSearchTerm, setDefaultReviewersState } = defaultReviewerProps
  const { getString } = useStrings()
  const setFieldValue = formik.setFieldValue

  const defaultReviewersList = useMemo(
    () =>
      settingSectionMode === SettingTypeMode.EDIT || formik.values.defaultReviewersSet
        ? formik.values.defaultReviewersList
        : [],
    [settingSectionMode, formik.values.defaultReviewersSet, formik.values.defaultReviewersList]
  )

  const minDefaultReviewers = formik.values.requireMinDefaultReviewers
  const defaultReviewersEnabled = formik.values.defaultReviewersEnabled
  const filteredPrincipalOptions = userPrincipalOptions.filter(
    (item: SelectOption) => !defaultReviewersList?.includes(item.value as string)
  )

  const defReviewerWarning = useMemo(() => {
    const minReviewers = Number(formik.values.minDefaultReviewers)
    const reviewerCount = defaultReviewersList?.length || 0

    if (formik.values.requireMinDefaultReviewers && minReviewers === reviewerCount) {
      let message = ''
      let showWarning = false

      if (reviewerCount === 1) {
        message = getString('protectionRules.defaultReviewerWarning')
        showWarning = true
      } else if (reviewerCount > 1) {
        message = getString('protectionRules.defaultReviewersWarning')
        showWarning = true
      }

      return { message, showWarning }
    }

    return { message: '', showWarning: false }
  }, [getString, formik.values, defaultReviewersList])

  return (
    <>
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.enableDefaultReviewersTitle')}
        name={'defaultReviewersEnabled'}
        onChange={e => {
          if (!(e.target as HTMLInputElement).checked) {
            setFieldValue('requireMinDefaultReviewers', false)
            formik.setFieldValue('defaultReviewersList', [])
          }
        }}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.enableDefaultReviewersText')}
      </Text>

      <Render when={defaultReviewersEnabled}>
        <Container padding={{ top: 'xlarge', left: 'xlarge' }}>
          <FormInput.Select
            items={filteredPrincipalOptions}
            onQueryChange={setSearchTerm}
            className={css.widthContainer}
            value={{ label: '', value: '' }}
            placeholder={getString('selectReviewers')}
            onChange={item => {
              const id = item.value?.toString().split(' ')[0]
              const displayName = item.label
              const defaultReviewerEntry = `${id} ${displayName}`
              defaultReviewersList?.push(defaultReviewerEntry)
              const uniqueArr = Array.from(new Set(defaultReviewersList))
              formik.setFieldValue('defaultReviewersList', uniqueArr)
              formik.setFieldValue('defaultReviewersSet', true)
              setDefaultReviewersState([...uniqueArr])
            }}
            name={'defaultReviewerSelect'}></FormInput.Select>
          {formik.errors.defaultReviewersList && (
            <Text color={Color.RED_350} padding={{ bottom: 'medium' }}>
              {formik.errors.defaultReviewersList}
            </Text>
          )}
          <Render when={defReviewerWarning.showWarning}>
            <Text color={Color.WARNING} padding={{ bottom: 'medium' }} style={{ width: '35%' }}>
              {defReviewerWarning.message}
            </Text>
          </Render>
          <DefaultReviewersList defaultReviewersList={defaultReviewersList} setFieldValue={formik.setFieldValue} />

          <FormInput.CheckBox
            className={css.checkboxLabel}
            label={getString('protectionRules.requireMinDefaultReviewersTitle')}
            name={'requireMinDefaultReviewers'}
            onChange={e => {
              if ((e.target as HTMLInputElement).checked) {
                setFieldValue('minDefaultReviewers', 1)
                setFieldValue('defaultReviewersEnabled', true)
              }
            }}
          />
          <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
            {getString('protectionRules.requireMinDefaultReviewersContent')}
          </Text>
          {minDefaultReviewers && (
            <Container padding={{ left: 'xlarge', top: 'medium' }}>
              <FormInput.Text
                className={cx(css.widthContainer, css.minText)}
                name={'minDefaultReviewers'}
                placeholder={getString('protectionRules.minNumberPlaceholder')}
                label={getString('protectionRules.minNumber')}
              />
            </Container>
          )}
        </Container>
      </Render>
      <hr className={css.dividerContainer} />
    </>
  )
}

export default DefaultReviewersSection
