import type { ForwardedRef } from 'react'
import { Range } from 'monaco-editor'
import type { editor, languages, Position } from 'monaco-editor'
import type { MonacoCodeEditorRef } from './SourceCodeEditorWithRef'
import type { YAMLSymbol } from './types'

export const setForwardedRef = <T>(ref: ForwardedRef<T>, value: T): void => {
  if (!ref) return

  if (typeof ref === 'function') {
    ref(value)
    return
  }

  ref.current = value
}

type CSSAttributes = { [key: string]: string }

export const highlightInsertedYAML = ({
  range,
  editor,
  style
}: {
  range: languages.DocumentSymbol['range']
  editor: MonacoCodeEditorRef
  style: CSSAttributes
}): NodeJS.Timeout => {
  const { endLineNumber } = range
  const pluginInputDecoration: editor.IModelDeltaDecoration = {
    range,
    options: {
      isWholeLine: false,
      className: style.highlight
    }
  }

  /* Scroll to the end of the inserted text */
  const endingLineNumber = endLineNumber > 0 ? endLineNumber - 1 : 0
  const endingColumnNumber = (editor.getModel()?.getLineContent(endingLineNumber) || '')?.length + 1
  editor.setPosition({ column: endingColumnNumber, lineNumber: endingLineNumber })
  editor.revealLineInCenter(endLineNumber)
  editor.focus()

  /* Add decorations */
  const decorations = editor.createDecorationsCollection([pluginInputDecoration])

  /* Auto-clear decorations */
  return setTimeout(() => {
    decorations.clear()
  }, 8000)
}

export const getStepCount = (symbols: YAMLSymbol[]): number => symbols.filter(symbol => symbol.name === 'steps').length

export function generateDefaultStepInsertionPath(stageIndex = 0): string {
  return `spec.stages.${stageIndex}.spec.steps`
}

export function* iterateSymbols(symbols: YAMLSymbol[], position: Position): Iterable<YAMLSymbol> {
  for (const symbol of symbols) {
    if (Range.containsPosition(symbol.range, position)) {
      yield symbol
      yield* iterateSymbols(symbol.children ? Object.keys(symbol.children) : [], position)
    }
  }
}
