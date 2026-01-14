/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { FC, PropsWithChildren } from 'react'
import type { IMenuItemProps } from '@blueprintjs/core'
import type { IconName, IconProps } from '@harnessio/icons'
import { Container, SplitButtonOption, Text } from '@harnessio/uicore'

import css from './CreateRepository.module.scss'

interface ButtonOptionProps extends Omit<IMenuItemProps, 'icon'> {
  subText?: React.ReactNode
  icon: IconName
  iconProps: Omit<IconProps, 'name'>
}

const ButtonOption: FC<PropsWithChildren<ButtonOptionProps>> = (props: ButtonOptionProps) => {
  const { icon, iconProps, children, ...rest } = props
  return (
    <SplitButtonOption
      {...rest}
      text={
        <Container>
          <Text icon={icon} iconProps={iconProps} className={css.optionLabel}>
            {props.text}
          </Text>
          {props.subText && (
            <Text className={css.optionSubLabel} lineClamp={3}>
              {props.subText}
            </Text>
          )}
        </Container>
      }>
      {children}
    </SplitButtonOption>
  )
}

export default ButtonOption
