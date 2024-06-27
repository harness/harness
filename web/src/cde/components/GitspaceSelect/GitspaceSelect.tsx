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

import React, { useEffect, useRef, useState } from 'react'
import cx from 'classnames'
import { Menu, PopoverInteractionKind, PopoverPosition } from '@blueprintjs/core'
import { Text, Button, Container, ButtonVariation, FormError } from '@harnessio/uicore'
import type { IconName } from '@harnessio/icons'
import { useFormikContext } from 'formik'
import { useStrings } from 'framework/strings'
import css from './GitspaceSelect.module.scss'

interface GitspaceSelectProps {
  text: React.ReactElement
  icon?: IconName
  renderMenu?: React.ReactElement
  disabled?: boolean
  overridePopOverWidth?: boolean
  errorMessage?: string
  formikName?: string
  tooltipProps?: { [key: string]: any }
}

export const GitspaceSelect = ({
  text,
  icon,
  renderMenu,
  disabled,
  overridePopOverWidth,
  errorMessage,
  formikName,
  tooltipProps
}: GitspaceSelectProps) => {
  const { getString } = useStrings()
  const buttonRef = useRef<HTMLDivElement | null>(null)
  const [popoverWidth, setPopoverWidth] = useState(0)

  const { touched } = useFormikContext<{ validated?: boolean }>()

  const defaultTooltipProps = {
    tooltip: (
      <Container className={css.listContainer} width={overridePopOverWidth ? '100%' : popoverWidth}>
        {renderMenu ? (
          renderMenu
        ) : (
          <Menu>
            <Text padding="small">{getString('cde.noData')}</Text>
          </Menu>
        )}
      </Container>
    ),
    tooltipProps: {
      fill: true,
      interactionKind: PopoverInteractionKind.CLICK,
      position: PopoverPosition.BOTTOM_LEFT,
      popoverClassName: cx(css.popover),
      ...tooltipProps
    }
  }

  useEffect(() => {
    if (
      buttonRef?.current?.getBoundingClientRect()?.width &&
      buttonRef?.current?.getBoundingClientRect()?.width !== popoverWidth
    ) {
      setPopoverWidth(buttonRef?.current?.getBoundingClientRect()?.width)
    }
  }, [buttonRef?.current, popoverWidth])

  const iconProp = icon ? { icon: icon as IconName } : {}

  const addTooltipProps = disabled ? {} : { ...defaultTooltipProps }

  return (
    <div className={css.buttonDiv} ref={buttonRef}>
      <Button
        className={cx(css.button, { [css.buttonWithoutIcon]: !icon })}
        text={text}
        rightIcon="chevron-down"
        variation={ButtonVariation.TERTIARY}
        iconProps={{ size: 14 }}
        {...iconProp}
        {...addTooltipProps}
        disabled={disabled}
      />
      {touched.validated && <FormError errorMessage={errorMessage} name={formikName || ''} />}
    </div>
  )
}
