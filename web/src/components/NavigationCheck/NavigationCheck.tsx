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
