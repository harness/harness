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
import { Container, Layout, Text } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import css from './LoadingSpinner.module.scss'

interface LoadingSpinnerProps {
  visible: boolean | null | undefined
  withBorder?: boolean
  className?: string
}

export const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({ visible, withBorder, className }) => {
  const { getString } = useStrings()

  return visible ? (
    <Container className={cx(css.main, { [css.withBorder]: withBorder }, className)}>
      <Layout.Vertical spacing="medium" className={css.layout}>
        <Icon name="steps-spinner" size={32} color={Color.GREY_600} />
        <Text font={{ size: 'medium', align: 'center' }} color={Color.GREY_600} className={css.text}>
          {getString('pageLoading')}
        </Text>
      </Layout.Vertical>
    </Container>
  ) : null
}
