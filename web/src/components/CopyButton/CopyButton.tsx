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

import React, { useEffect, useState } from 'react'
import { Button, Utils, ButtonProps, ButtonVariation } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'

interface CopyButtonProps extends ButtonProps {
  content: string
}

export function CopyButton({ content, icon, iconProps, ...props }: CopyButtonProps) {
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    let timeoutId: number
    if (copied) {
      timeoutId = window.setTimeout(() => setCopied(false), 2500)
    }
    return () => {
      clearTimeout(timeoutId)
    }
  }, [copied])

  return (
    <Button
      variation={ButtonVariation.ICON}
      icon={copied ? 'tick' : icon || 'copy-alt'}
      iconProps={{ color: copied ? Color.GREEN_500 : undefined, ...iconProps }}
      onClick={async () => {
        setCopied(true)
        await Utils.copy(content)
      }}
      {...props}
    />
  )
}
