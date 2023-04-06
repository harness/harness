import React, { useEffect, useMemo, useRef } from 'react'
import { Container } from '@harness/uicore'
import { LanguageDescription } from '@codemirror/language'
import { indentWithTab } from '@codemirror/commands'
import type { ViewUpdate } from '@codemirror/view'
import { languages } from '@codemirror/language-data'
import { EditorView, keymap } from '@codemirror/view'
import { noop } from 'lodash-es'
import { Compartment, EditorState, Extension } from '@codemirror/state'
import { color } from '@uiw/codemirror-extensions-color'
import { hyperLink } from '@uiw/codemirror-extensions-hyper-link'
import { githubLight as theme } from '@uiw/codemirror-themes-all'

interface EditorProps {
  filename: string
  source: string
  onViewUpdate?: (update: ViewUpdate) => void
  readonly?: boolean
  className?: string
  extensions?: Extension
  viewRef?: React.MutableRefObject<EditorView | undefined>
}

export const Editor = React.memo(function CodeMirrorReactEditor({
  source,
  filename,
  onViewUpdate = noop,
  readonly = false,
  className,
  extensions = new Compartment().of([]),
  viewRef
}: EditorProps) {
  const view = useRef<EditorView>()
  const ref = useRef<HTMLDivElement>()
  const languageConfig = useMemo(() => new Compartment(), [])

  useEffect(() => {
    const editorView = new EditorView({
      doc: source,
      extensions: [
        extensions,

        color,
        hyperLink,
        theme,

        EditorView.lineWrapping,
        keymap.of([indentWithTab]),

        ...(readonly ? [EditorState.readOnly.of(true), EditorView.editable.of(false)] : []),

        EditorView.updateListener.of(onViewUpdate),

        /**
        languageConfig is a compartment that defaults to an empty array (no language support)
        at first, when a language is detected, languageConfig is used to reconfigure dynamically.
        @see https://codemirror.net/examples/config/
        */
        languageConfig.of([])
      ],
      parent: ref.current
    })

    view.current = editorView

    if (viewRef) {
      viewRef.current = editorView
    }

    return () => {
      editorView.destroy()
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Dynamically load language support based on filename
  useEffect(() => {
    if (filename) {
      languageDescriptionFrom(filename)
        ?.load()
        .then(languageSupport => {
          view.current?.dispatch({ effects: languageConfig.reconfigure(languageSupport) })
        })
    }
  }, [filename, view, languageConfig])

  return <Container ref={ref} className={className} />
})

function languageDescriptionFrom(filename: string) {
  return LanguageDescription.matchFilename(languages, filename)
}
