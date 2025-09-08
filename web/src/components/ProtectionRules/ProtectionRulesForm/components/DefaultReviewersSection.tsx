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
import { Container, FormInput, Layout, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import type { FormikProps } from 'formik'
import { Render } from 'react-jsx-match'
import { useStrings } from 'framework/strings'
import {
  getFilteredNormalizedPrincipalOptions,
  type RulesFormPayload
} from 'components/ProtectionRules/ProtectionRulesUtils'
import { NormalizedPrincipal, PrincipalType } from 'utils/Utils'
import SearchDropDown, { renderPrincipalIcon } from 'components/SearchDropDown/SearchDropDown'
import NormalizedPrincipalsList from './NormalizedPrincipalsList'
import css from '../ProtectionRulesForm.module.scss'

export interface DefaultReviewerProps {
  loading: boolean
  searchTerm: string
  setSearchTerm: React.Dispatch<React.SetStateAction<string>>
  normalizedPrincipalOptions: NormalizedPrincipal[]
}

const DefaultReviewersSection = ({
  formik,
  defaultReviewerProps
}: {
  formik: FormikProps<RulesFormPayload>
  defaultReviewerProps: DefaultReviewerProps
}) => {
  const { loading, normalizedPrincipalOptions, searchTerm, setSearchTerm } = defaultReviewerProps
  const { getString } = useStrings()
  const setFieldValue = formik.setFieldValue
  const { defaultReviewersEnabled, defaultReviewersList, requireMinDefaultReviewers } = formik.values
  const { defaultReviewersList: formikDefaultReviewersListError } = formik.errors

  return (
    <>
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.enableDefaultReviewersTitle')}
        name={'defaultReviewersEnabled'}
        onChange={e => {
          if (!(e.target as HTMLInputElement).checked) {
            setFieldValue('requireMinDefaultReviewers', false)
            setFieldValue('defaultReviewersList', [])
          }
        }}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.enableDefaultReviewersText')}
      </Text>

      <Render when={defaultReviewersEnabled}>
        <Container padding={{ top: 'xlarge', left: 'xlarge' }}>
          <SearchDropDown
            searchTerm={searchTerm}
            placeholder={getString('selectUsersAndUserGroups')}
            className={css.widthContainer}
            onChange={setSearchTerm}
            options={getFilteredNormalizedPrincipalOptions(
              normalizedPrincipalOptions.filter(user => user.type !== PrincipalType.SERVICE_ACCOUNT),
              defaultReviewersList || []
            )}
            loading={loading}
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
              const updatedList = [value, ...(defaultReviewersList || [])]
              const uniqueArr = Array.from(new Map(updatedList.map(item => [item.id, item])).values())
              formik.setFieldValue('defaultReviewersList', uniqueArr)
            }}
          />
          {formikDefaultReviewersListError && <Text color={Color.RED_350}>{formikDefaultReviewersListError}</Text>}

          <NormalizedPrincipalsList
            fieldName={'defaultReviewersList'}
            list={defaultReviewersList}
            setFieldValue={formik.setFieldValue}
          />

          <Container padding={{ top: 'large' }}>
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
          </Container>

          {requireMinDefaultReviewers && (
            <Container padding={{ left: 'xlarge', top: 'medium' }}>
              <FormInput.Text
                inputGroup={{ type: 'number' }}
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
