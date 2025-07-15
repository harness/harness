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
import type { PrincipalType } from 'utils/Utils'
import type { NormalizedPrincipal } from 'components/ProtectionRules/ProtectionRulesUtils'
import css from '../ProtectionRulesForm.module.scss'

const BypassList = (props: {
  renderPrincipalIcon: (type: PrincipalType, displayName: string) => JSX.Element
  bypassList?: NormalizedPrincipal[]
  setFieldValue: (field: string, value: any, shouldValidate?: boolean) => void
}) => {
  const { bypassList, setFieldValue, renderPrincipalIcon } = props

  return (
    <Container>
      {!isEmpty(bypassList) && (
        <Text color={Color.GREY_500} padding={{ top: 'medium', bottom: 'small' }} font={{ weight: 'semi-bold' }}>
          Bypass List ({bypassList?.length})
        </Text>
      )}
      <Container className={css.bypassContainer}>
        {bypassList?.map((userObj, idx: number) => {
          const { id, display_name, email_or_identifier, type } = userObj
          return (
            <Layout.Horizontal
              key={`${display_name}-${idx}-${id}-${email_or_identifier}`}
              flex={{ align: 'center-center' }}
              padding={{ right: 'small', left: 'small' }}>
              {renderPrincipalIcon(type as PrincipalType, display_name)}
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
                  const filteredData = bypassList.filter(item => !(item.id === id))
                  setFieldValue('bypassList', filteredData)
                }}
                className={css.codeClose}
              />
            </Layout.Horizontal>
          )
        })}
      </Container>
    </Container>
  )
}

export default BypassList
