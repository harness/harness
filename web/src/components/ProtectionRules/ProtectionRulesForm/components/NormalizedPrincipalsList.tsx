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
import { Icon } from '@harnessio/icons'
import { Container, FlexExpander, Layout, Text } from '@harnessio/uicore'
import { PopoverPosition } from '@blueprintjs/core'
import { isEmpty } from 'lodash-es'
import { Color } from '@harnessio/design-system'
import type { NormalizedPrincipal } from 'utils/Utils'
import { renderPrincipalIcon } from 'components/SearchDropDown/SearchDropDown'
import { StringKeys, useStrings } from 'framework/strings'
import css from '../ProtectionRulesForm.module.scss'

const NormalizedPrincipalsList = ({
  fieldName,
  list,
  setFieldValue
}: {
  fieldName: string
  list?: NormalizedPrincipal[]
  setFieldValue: (field: string, value: any, shouldValidate?: boolean) => void
}) => {
  const { getString } = useStrings()

  return (
    <>
      {!isEmpty(list) && (
        <Text color={Color.GREY_500} padding={{ top: 'medium', bottom: 'small' }} font={{ weight: 'semi-bold' }}>
          {getString(`protectionRules.${fieldName}` as StringKeys)} ({list?.length})
        </Text>
      )}
      <Container className={css.bypassContainer}>
        {list?.map((userObj, idx: number) => {
          const { id, display_name, email_or_identifier, type } = userObj
          return (
            <Layout.Horizontal
              key={`${display_name}-${idx}-${id}-${email_or_identifier}`}
              flex={{ align: 'center-center' }}
              padding={{ right: 'small', left: 'small' }}>
              {renderPrincipalIcon(type, display_name)}
              <Text
                style={{ cursor: 'pointer' }}
                tooltip={email_or_identifier}
                tooltipProps={{
                  interactionKind: 'hover',
                  position: PopoverPosition.TOP
                }}>
                {display_name}
              </Text>
              <FlexExpander />
              <Icon
                name="code-close"
                onClick={() => {
                  const filteredData = list.filter(item => !(item.id === id))
                  setFieldValue(fieldName, filteredData)
                }}
                className={css.codeClose}
              />
            </Layout.Horizontal>
          )
        })}
      </Container>
    </>
  )
}

export default NormalizedPrincipalsList
