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

import React, { useState } from 'react'
import copy from 'clipboard-copy'
import { Color } from '@harnessio/design-system'
import { Button, ButtonProps, ButtonVariation } from '@harnessio/uicore'
import { useStrings } from '@ar/frameworks/strings/String'

interface CopyButtonProps extends ButtonProps {
  textToCopy: string
  onCopySuccess?: (evt: React.MouseEvent<Element, MouseEvent>) => void
  primaryBtn?: boolean
}

const CopyButton: React.FC<CopyButtonProps> = ({ textToCopy, onCopySuccess, ...rest }): JSX.Element => {
  const { getString } = useStrings()
  const [openTooltip, setOpenTooltip] = useState(false)
  const showCopySuccess = (): void => {
    setOpenTooltip(true)
    setTimeout(
      /* istanbul ignore next */ () => {
        setOpenTooltip(false)
      },
      1000
    )
  }

  return (
    <Button
      variation={rest.primaryBtn ? ButtonVariation.PRIMARY : undefined}
      minimal
      iconProps={{ color: rest.primaryBtn ? Color.WHITE : undefined }}
      icon="duplicate"
      onClick={evt => {
        copy(textToCopy)
        showCopySuccess()
        if (onCopySuccess) {
          onCopySuccess(evt)
        }
      }}
      withoutCurrentColor
      tooltip={getString('copied')}
      tooltipProps={{ isOpen: openTooltip, isDark: true }}
      {...rest}
    />
  )
}

export default CopyButton
