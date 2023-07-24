import React from 'react'
import { Intent } from '@blueprintjs/core'
import { useCallback, useRef, useState } from 'react'
import { noop } from 'lodash-es'
import { useConfirmationDialog, Text } from '@harness/uicore'
import { useStrings } from 'framework/strings'

export interface UseConfirmActionDialogProps {
  message: React.ReactElement
  intent?: Intent
  title?: string
  confirmText?: string
  cancelText?: string
  action: (params?: Unknown) => void
}

/**
 * @deprecated Use useConfirmAct() hook instead
 */
export const useConfirmAction = (props: UseConfirmActionDialogProps) => {
  const { title, message, confirmText, cancelText, intent, action } = props
  const { getString } = useStrings()
  const [params, setParams] = useState<Unknown>()
  const { openDialog } = useConfirmationDialog({
    intent,
    titleText: title || getString('confirmation'),
    contentText: message,
    confirmButtonText: confirmText || getString('confirm'),
    cancelButtonText: cancelText || getString('cancel'),
    buttonIntent: intent || Intent.DANGER,
    onCloseDialog: async (isConfirmed: boolean) => {
      if (isConfirmed) {
        action(params)
      }
    }
  })
  const confirm = useCallback(
    (_params?: Unknown) => {
      setParams(_params)
      openDialog()
    },
    [openDialog]
  )

  return confirm
}

interface ConfirmActArgs {
  title?: string
  message: React.ReactNode
  intent?: Intent
  confirmText?: string
  cancelText?: string
  action: () => Promise<void> | void
}

export const useConfirmAct = () => {
  const { getString } = useStrings()
  const [_args, setArgs] = useState<ConfirmActArgs>({ message: '', action: noop })
  const resolve = useRef<() => void>(noop)
  const { openDialog } = useConfirmationDialog({
    titleText: _args.title || getString('confirmation'),
    contentText: toParagraph(_args.message),
    intent: _args.intent,
    confirmButtonText: _args.confirmText || getString('confirm'),
    cancelButtonText: _args.cancelText || getString('cancel'),
    buttonIntent: _args.intent || Intent.DANGER,
    onCloseDialog: async (isConfirmed: boolean) => {
      if (isConfirmed) {
        await _args.action()
      }
      resolve.current()
    }
  })

  return useCallback(
    async (args: ConfirmActArgs) => {
      setArgs({ ..._args, ...args })
      openDialog()
      return new Promise<void>(_resolve => (resolve.current = _resolve))
    },
    [_args, openDialog]
  )
}

function toParagraph(message: React.ReactNode) {
  if (typeof message === 'string' && message.includes('\\n')) {
    return (
      <>
        {message.split('\\n').map((line, i) => (
          <Text key={i} tag="span" style={{ display: 'block', marginTop: '10px', wordBreak: 'break-word' }}>
            {line}
          </Text>
        ))}
      </>
    )
  }

  return message
}
