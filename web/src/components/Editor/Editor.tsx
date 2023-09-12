import React, { useEffect, useMemo, useRef } from 'react'
import { Container } from '@harnessio/uicore'
import { LanguageDescription } from '@codemirror/language'
import { indentWithTab } from '@codemirror/commands'
import cx from 'classnames'
import type { ViewUpdate } from '@codemirror/view'
import type { Text } from '@codemirror/state'
import { languages } from '@codemirror/language-data'
import { markdown } from '@codemirror/lang-markdown'
import { EditorView, keymap, placeholder as placeholderExtension } from '@codemirror/view'
import { Compartment, EditorState, Extension } from '@codemirror/state'
import { color } from '@uiw/codemirror-extensions-color'
import { hyperLink } from '@uiw/codemirror-extensions-hyper-link'
import { githubLight, githubDark } from '@uiw/codemirror-themes-all'
import css from './Editor.module.scss'

export interface EditorProps {
  content: string
  filename?: string
  forMarkdown?: boolean
  placeholder?: string
  readonly?: boolean
  autoFocus?: boolean
  className?: string
  extensions?: Extension
  maxHeight?: string | number
  viewRef?: React.MutableRefObject<EditorView | undefined>
  setDirty?: React.Dispatch<React.SetStateAction<boolean>>
  onChange?: (doc: Text, viewUpdate: ViewUpdate, isDirty: boolean) => void
  onViewUpdate?: (viewUpdate: ViewUpdate) => void
  darkTheme?: boolean
}

export const Editor = React.memo(function CodeMirrorReactEditor({
  content,
  filename,
  forMarkdown,
  placeholder,
  readonly = false,
  autoFocus,
  className,
  extensions = new Compartment().of([]),
  maxHeight,
  viewRef,
  setDirty,
  onChange,
  onViewUpdate,
  darkTheme
}: EditorProps) {
  const contentRef = useRef(content)
  const view = useRef<EditorView>()
  const ref = useRef<HTMLDivElement>()
  const languageConfig = useMemo(() => new Compartment(), [])
  const markdownLanguageSupport = useMemo(() => markdown({ codeLanguages: languages }), [])
  const style = useMemo(() => {
    if (maxHeight) {
      return {
        '--editor-max-height': Number.isInteger(maxHeight) ? `${maxHeight}px` : maxHeight
      } as React.CSSProperties
    }
  }, [maxHeight])
  const onChangeRef = useRef<EditorProps['onChange']>(onChange)

  useEffect(() => {
    onChangeRef.current = onChange
  }, [onChange])

  useEffect(() => {
    const editorView = new EditorView({
      doc: content,
      extensions: [
        extensions,

        color,
        hyperLink,
        darkTheme ? githubDark : githubLight,

        EditorView.lineWrapping,

        ...(placeholder ? [placeholderExtension(placeholder)] : []),

        keymap.of([indentWithTab]),

        ...(readonly ? [EditorState.readOnly.of(true), EditorView.editable.of(false)] : []),

        EditorView.updateListener.of(viewUpdate => {
          const isDirty = !cleanDoc.eq(viewUpdate.state.doc)
          setDirty?.(isDirty)
          onViewUpdate?.(viewUpdate)

          if (viewUpdate.docChanged) {
            onChangeRef.current?.(viewUpdate.state.doc, viewUpdate, isDirty)
          }
        }),

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

    const cleanDoc = editorView.state.doc

    if (autoFocus) {
      editorView.focus()
    }

    return () => {
      editorView.destroy()
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Dynamically load language support based on filename. Note that
  // we need to configure languageSupport for Markdown separately to
  // enable syntax highlighting for all code blocks (multi-lang).
  useEffect(() => {
    if (forMarkdown) {
      view.current?.dispatch({ effects: languageConfig.reconfigure(markdownLanguageSupport) })
    } else if (filename) {
      LanguageDescription.matchFilename(languages, filename)
        ?.load()
        .then(languageSupport => {
          view.current?.dispatch({
            effects: languageConfig.reconfigure(
              languageSupport.language.name === 'markdown' ? markdownLanguageSupport : languageSupport
            )
          })
        })
    }
  }, [filename, forMarkdown, view, languageConfig, markdownLanguageSupport])

  useEffect(() => {
    if (contentRef.current !== content) {
      contentRef.current = content
      viewRef?.current?.dispatch({
        changes: { from: 0, to: viewRef?.current?.state.doc.length, insert: content }
      })
    }
  }, [content, viewRef])

  return <Container ref={ref} className={cx(css.editor, className)} style={style} />
})
