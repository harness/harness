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
import { Icon } from '@harnessio/icons'
import { Avatar, Container, FlexExpander, Layout, Text } from '@harnessio/uicore'
import css from '../ProtectionRulesForm.module.scss'

const BypassList = (props: {
  bypassList?: string[] // eslint-disable-next-line @typescript-eslint/no-explicit-any
  setFieldValue: (field: string, value: any, shouldValidate?: boolean) => void
}) => {
  const { bypassList, setFieldValue } = props

  const bypassContent = useMemo(() => {
    return (
      <Container className={cx(css.widthContainer, css.bypassContainer)}>
        {bypassList?.map((owner: string, idx: number) => {
          const name = owner.slice(owner.indexOf(' ') + 1)
          return (
            <Layout.Horizontal key={`${name}-${idx}`} flex={{ align: 'center-center' }} padding={'small'}>
              <Avatar hoverCard={false} size="small" name={name.toString()} />
              <Text padding={{ top: 'tiny' }} lineClamp={1}>
                {name}
              </Text>
              <FlexExpander />
              <Icon
                name="code-close"
                onClick={() => {
                  const filteredData = bypassList.filter(item => !(item === owner))
                  setFieldValue('bypassList', filteredData)
                }}
                className={css.codeClose}
              />
            </Layout.Horizontal>
          )
        })}
      </Container>
    )
  }, [bypassList, setFieldValue])

  return bypassContent
}

export default BypassList
