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

import React, { useState, useEffect, useCallback } from 'react'
import { Prompt, useHistory } from 'react-router-dom'
import { Intent } from '@harnessio/design-system'
import type * as History from 'history'
import { useStrings } from 'framework/strings'
import { useConfirmationDialog } from 'hooks/useConfirmationDialog'
import css from './NavigationCheck.module.scss'

interface NavigationCheckProps {
  when?: boolean
  i18n?: {
    message?: string
    title?: string
    confirmText?: string
    cancelText?: string
  }
  replace?: boolean
}

export const NavigationCheck: React.FC<NavigationCheckProps> = ({ when, i18n, replace }) => {
  const history = useHistory()
  const [lastLocation, setLastLocation] = useState<History.Location | null>(null)
  const [confirmed, setConfirmed] = useState(false)
  const { getString } = useStrings()
  const { openDialog } = useConfirmationDialog({
    cancelButtonText: i18n?.cancelText || getString('unsavedChanges.stay'),
    showCloseButton: false,
    contentText: i18n?.message || getString('unsavedChanges.message'),
    titleText: i18n?.title || getString('unsavedChanges.title'),
    confirmButtonText: i18n?.confirmText || getString('unsavedChanges.leave'),
    intent: Intent.WARNING,
    onCloseDialog: setConfirmed,
    className: css.main
  })
  const prompt = useCallback(
    (nextLocation: History.Location): string | boolean => {
      if (!confirmed) {
        openDialog()
        setLastLocation(nextLocation)
        return false
      }
      return true
    },
    [confirmed, openDialog]
  )

  useEffect(() => {
    if (confirmed && lastLocation) {
      history[replace ? 'replace' : 'push'](lastLocation.pathname + lastLocation.search)
    }

    confirmed && setConfirmed(false)
  }, [confirmed, lastLocation, history, replace])

  return <Prompt when={when} message={prompt} />
}
