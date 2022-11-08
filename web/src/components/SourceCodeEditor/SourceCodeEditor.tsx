import React, { lazy, Suspense } from 'react'
import { Text } from '@harness/uicore'
import type { SourceCodeEditorProps } from 'utils/Utils'
import { useStrings } from 'framework/strings'

function Editor(props: SourceCodeEditorProps) {
  const { getString } = useStrings()
  const MonacoSourceCodeEditor = lazy(() => import('./MonacoSourceCodeEditor'))

  return (
    <Suspense fallback={<Text>{getString('loading')}</Text>}>
      <MonacoSourceCodeEditor {...props} />
    </Suspense>
  )
}

export const SourceCodeEditor = React.memo(Editor)
