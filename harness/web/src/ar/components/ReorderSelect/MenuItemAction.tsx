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

import React from 'react'
import type { IconName } from '@harnessio/icons'
import { Button, ButtonSize } from '@harnessio/uicore'
import css from './ReorderSelect.module.scss'

interface MenuItemActionProps {
  onClick?: (event: React.MouseEvent<Element, MouseEvent>) => void
  rightIcon?: IconName | undefined
  icon?: IconName | undefined
  text?: string | React.ReactNode
}

function MenuItemAction({ onClick, rightIcon, icon, text }: MenuItemActionProps): JSX.Element {
  return (
    <Button
      className={css.addBtn}
      icon={icon}
      rightIcon={rightIcon}
      iconProps={{ size: 12 }}
      minimal
      intent="primary"
      onClick={onClick}
      text={text}
      size={ButtonSize.SMALL}
    />
  )
}

export default MenuItemAction
