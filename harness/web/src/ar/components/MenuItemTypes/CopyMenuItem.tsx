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
import copy from 'clipboard-copy'
import { useToaster } from '@harnessio/uicore'

import { useParentComponents } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

interface CopyMenuItemProps {
  value: string
  onCopy?: () => void
}

function CopyMenuItem(props: CopyMenuItemProps): JSX.Element {
  const { value, onCopy } = props
  const { RbacMenuItem } = useParentComponents()
  const { getString } = useStrings()
  const { showSuccess } = useToaster()

  const handleCopy = () => {
    copy(value)
    onCopy?.()
    showSuccess(getString('copied'))
  }
  return <RbacMenuItem icon="code-copy" text={getString('actions.copyCommand')} onClick={handleCopy} />
}

export default CopyMenuItem
