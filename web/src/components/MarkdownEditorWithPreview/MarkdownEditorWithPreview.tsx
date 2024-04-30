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

import React, { useCallback, useEffect, useRef, useState } from 'react'
import {
  Text,
  Button,
  Container,
  ButtonVariation,
  Layout,
  ButtonSize,
  Dialog,
  FlexExpander,
  useToaster
} from '@harnessio/uicore'
import type { IconName } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import cx from 'classnames'
import { DecorationSet, EditorView, Decoration, keymap } from '@codemirror/view'
import { undo, redo, history } from '@codemirror/commands'
import { EditorSelection, StateEffect, StateField } from '@codemirror/state'
import { isEmpty, isString } from 'lodash-es'
import { useMutate } from 'restful-react'
import { useAppContext } from 'AppContext'
import { Editor } from 'components/Editor/Editor'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
import { useStrings } from 'framework/strings'
import {
  CommentBoxOutletPosition,
  formatBytes,
  getErrorMessage,
  handleFileDrop,
  handlePaste,
  removeSpecificTextOptimized
} from 'utils/Utils'
import { decodeGitContent, handleUpload, normalizeGitRef } from 'utils/GitUtils'
import { defaultUsefulOrNot } from 'components/DefaultUsefulOrNot/UsefulOrNot'
import { AidaClient } from 'utils/types'
import type { TypesRepository } from 'services/code'
import { useEventListener } from 'hooks/useEventListener'
import css from './MarkdownEditorWithPreview.module.scss'

enum MarkdownEditorTab {
  WRITE = 'write',
  PREVIEW = 'preview'
}

enum ToolbarAction {
  HEADER = 'HEADER',
  BOLD = 'BOLD',
  ITALIC = 'ITALIC',
  UPLOAD = 'UPLOAD',
  UNORDER_LIST = 'UNORDER_LIST',
  CHECK_LIST = 'CHECK_LIST',
  CODE_BLOCK = 'CODE_BLOCK'
}

interface ToolbarItem {
  icon: IconName
  action: ToolbarAction
}

const toolbar: ToolbarItem[] = [
  { icon: 'header', action: ToolbarAction.HEADER },
  { icon: 'bold', action: ToolbarAction.BOLD },
  { icon: 'italic', action: ToolbarAction.ITALIC },
  { icon: 'paperclip', action: ToolbarAction.UPLOAD },

  { icon: 'properties', action: ToolbarAction.UNORDER_LIST },
  { icon: 'form', action: ToolbarAction.CHECK_LIST },
  { icon: 'main-code-yaml', action: ToolbarAction.CODE_BLOCK }
]

// Define a unique effect to update decorations
const addDecorationEffect = StateEffect.define<{ decoration: Decoration; from: number; to: number }[]>()
const removeDecorationEffect = StateEffect.define<{}>() // No payload needed for removal in this simple case// Create a state field to hold decorations
const decorationField = StateField.define<DecorationSet>({
  create() {
    return Decoration.none
  },
  update(decorations, tr) {
    decorations = decorations.map(tr.changes)

    for (const effect of tr.effects) {
      if (effect.is(addDecorationEffect)) {
        const add = []
        for (const { decoration, from, to } of effect.value) {
          add.push(decoration.range(from, to))
        }
        decorations = decorations.update({ add })
      } else if (effect.is(removeDecorationEffect)) {
        decorations = Decoration.none
      }
    }
    return decorations
  },
  provide: f => EditorView.decorations.from(f)
})

interface MarkdownEditorWithPreviewProps {
  className?: string
  value?: string
  templateData?: string
  onChange?: (value: string) => void
  onSave?: (value: string) => void
  onCancel?: () => void
  setDirty?: (dirty: boolean) => void
  i18n: {
    placeHolder: string
    tabEdit: string
    tabPreview: string
    cancel: string
    save: string
  }
  hideButtons?: boolean
  hideCancel?: boolean
  editorHeight?: string
  noBorder?: boolean
  viewRef?: React.MutableRefObject<EditorView | undefined>
  secondarySaveButton?: typeof Button

  // When set to true, the editor will be scrolled to center of screen
  // and cursor is set to the end of the document
  autoFocusAndPosition?: boolean
  outlets?: Partial<Record<CommentBoxOutletPosition, React.ReactNode>>
  handleCopilotClick?: () => void
  flag?: boolean
  sourceGitRef?: string
  targetGitRef?: string
  setFlag?: React.Dispatch<React.SetStateAction<boolean>>
  repoMetadata: TypesRepository | undefined
  standalone: boolean
  routingId: string
}

export function MarkdownEditorWithPreview({
  className,
  value = '',
  templateData = '',
  onChange,
  onSave,
  onCancel,
  setDirty: setDirtyProp,
  i18n,
  hideButtons,
  hideCancel,
  editorHeight,
  noBorder,
  viewRef: viewRefProp,
  autoFocusAndPosition,
  secondarySaveButton: SecondarySaveButton,
  repoMetadata,
  standalone,
  routingId,
  handleCopilotClick,
  outlets = {},
  setFlag,
  flag,
  sourceGitRef,
  targetGitRef
}: MarkdownEditorWithPreviewProps) {
  const { getString } = useStrings()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [selectedTab, setSelectedTab] = useState(MarkdownEditorTab.WRITE)
  const viewRef = useRef<EditorView>()
  const containerRef = useRef<HTMLDivElement>(null)
  const [dirty, setDirty] = useState(false)
  const [open, setOpen] = useState(false)
  const [file, setFile] = useState<File>()
  const { showError } = useToaster()
  const [markdownContent, setMarkdownContent] = useState('')
  const { customComponents } = useAppContext()
  const AIDAFeedback = customComponents?.UsefulOrNot ? customComponents.UsefulOrNot : defaultUsefulOrNot
  const { mutate } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata?.path}/+/genai/change-summary`
  })
  const isDirty = useRef(dirty)
  const [data, setData] = useState({})

  useEffect(
    function setDirtyRef() {
      isDirty.current = dirty
    },
    [dirty]
  )

  const myKeymap = keymap.of([
    {
      key: 'Mod-z',
      run: undo,
      preventDefault: true
    },
    { key: 'Mod-Shift-z', run: redo, preventDefault: true },
    {
      key: 'Mod-Enter',
      run: () => {
        if (isDirty.current) onSaveHandler()
        return true
      },
      preventDefault: true
    }
  ])
  const handleMouseDown = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (event: any) => {
      const editorDom = viewRef?.current?.dom
      if (!editorDom?.contains(event.target)) {
        // Clicked outside the editor
        viewRef?.current?.dispatch({
          effects: removeDecorationEffect.of({})
        })
      }
    },
    [viewRef]
  )

  useEventListener('mousedown', handleMouseDown)
  const [generating, setGenerating] = useState<boolean>(false)
  const dispatchContent = (content: string, userEvent: boolean, decoration = false) => {
    const view = viewRef.current
    const { from, to } = view?.state.selection.main ?? { from: 0, to: 0 }
    const changeColorDecoration = Decoration.mark({ class: 'aidaGenText' })
    const highlightDecoration = Decoration.mark({ class: 'highlightText' })

    if (decoration) {
      view?.dispatch({
        changes: { from: from, to: to, insert: content },
        effects: addDecorationEffect.of([
          { decoration: changeColorDecoration, from: from, to: content?.length + from }
        ]),
        userEvent: userEvent ? 'input' : 'ignore' // Marking this transaction as an input event makes it part of the undo history,
      })
    } else {
      view?.dispatch({
        effects: removeDecorationEffect.of({ from: from, to: to }),
        userEvent: userEvent ? 'input' : 'ignore' // Marking this transaction as an input event makes it part of the undo history
      })
      view?.dispatch({
        changes: { from: from, to: to, insert: content },
        effects: addDecorationEffect.of([{ decoration: highlightDecoration, from: from, to: content?.length + from }]),
        userEvent: userEvent ? 'input' : 'ignore' // Marking this transaction as an input event makes it part of the undo history
      })
      removeSpecificTextOptimized(viewRef, getString('aidaGenSummary'))
    }
  }

  useEffect(() => {
    if (flag) {
      if (handleCopilotClick) {
        setGenerating(true)
        dispatchContent(getString('aidaGenSummary'), false, true)
        mutate({
          head_ref: normalizeGitRef(sourceGitRef),
          base_ref: normalizeGitRef(targetGitRef)
        })
          .then(res => {
            setGenerating(false)
            setData(res.summary || '')
          })
          .catch(err => {
            showError(getErrorMessage(err))
          })
      }
      setFlag?.(false)
    }
  }, [handleCopilotClick, flag]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!isEmpty(data)) {
      dispatchContent(`${data}`, true)
    }
  }, [data]) // eslint-disable-line react-hooks/exhaustive-deps
  const onToolbarAction = useCallback((action: ToolbarAction) => {
    const view = viewRef.current

    if (!view?.state) {
      return
    }

    // Note: Part of this code is copied from @uiwjs/react-markdown-editor
    // MIT License, Copyright (c) 2020 uiw
    // @see https://github.dev/uiwjs/react-markdown-editor/blob/2d3f45079c79616b867ef03681a8ba9799169921/src/commands/header.tsx
    switch (action) {
      case ToolbarAction.HEADER: {
        const lineInfo = view.state.doc.lineAt(view.state.selection.main.from)
        let mark = '#'
        const matchMark = lineInfo.text.match(/^#+/)
        if (matchMark && matchMark[0]) {
          const txt = matchMark[0]
          if (txt.length < 6) {
            mark = txt + '#'
          }
        }
        if (mark.length > 6) {
          mark = '#'
        }
        const title = lineInfo.text.replace(/^#+/, '')
        view.dispatch({
          changes: {
            from: lineInfo.from,
            to: lineInfo.to,
            insert: `${mark} ${title}`
          },
          // selection: EditorSelection.range(lineInfo.from + mark.length, lineInfo.to),
          selection: { anchor: lineInfo.from + mark.length + 1 }
        })
        break
      }

      case ToolbarAction.UPLOAD: {
        setFile(undefined)
        setOpen(true)
        break
      }

      case ToolbarAction.BOLD: {
        view.dispatch(
          view.state.changeByRange(range => ({
            changes: [
              { from: range.from, insert: '**' },
              { from: range.to, insert: '**' }
            ],
            range: EditorSelection.range(range.from + 2, range.to + 2)
          }))
        )
        break
      }

      case ToolbarAction.ITALIC: {
        view.dispatch(
          view.state.changeByRange(range => ({
            changes: [
              { from: range.from, insert: '*' },
              { from: range.to, insert: '*' }
            ],
            range: EditorSelection.range(range.from + 1, range.to + 1)
          }))
        )
        break
      }

      case ToolbarAction.UNORDER_LIST: {
        const lineInfo = view.state.doc.lineAt(view.state.selection.main.from)
        let mark = '- '
        const matchMark = lineInfo.text.match(/^-/)
        if (matchMark && matchMark[0]) {
          mark = ''
        }
        view.dispatch({
          changes: {
            from: lineInfo.from,
            to: lineInfo.to,
            insert: `${mark}${lineInfo.text}`
          },
          // selection: EditorSelection.range(lineInfo.from + mark.length, lineInfo.to),
          selection: { anchor: view.state.selection.main.from + mark.length }
        })
        break
      }

      case ToolbarAction.CHECK_LIST: {
        const lineInfo = view.state.doc.lineAt(view.state.selection.main.from)
        let mark = '- [ ]  '
        const matchMark = lineInfo.text.match(/^-\s\[\s\]\s/)
        if (matchMark && matchMark[0]) {
          mark = ''
        }
        view.dispatch({
          changes: {
            from: lineInfo.from,
            to: lineInfo.to,
            insert: `${mark}${lineInfo.text}`
          },
          // selection: EditorSelection.range(lineInfo.from + mark.length, lineInfo.to),
          selection: { anchor: view.state.selection.main.from + mark.length }
        })
        break
      }

      case ToolbarAction.CODE_BLOCK: {
        const main = view.state.selection.main
        const txt = view.state.sliceDoc(view.state.selection.main.from, view.state.selection.main.to)
        view.dispatch({
          changes: {
            from: main.from,
            to: main.to,
            insert: `\`\`\`tsx\n${txt}\n\`\`\``
          },
          selection: EditorSelection.range(main.from + 3, main.from + 6)
        })
        break
      }
    }
  }, [])

  useEffect(() => {
    setDirtyProp?.(dirty)

    return () => {
      setDirtyProp?.(false)
    }
  }, [dirty]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (viewRefProp) {
      viewRefProp.current = viewRef.current
    }
  }, [viewRefProp, viewRef.current]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (autoFocusAndPosition && !dirty) {
      scrollToAndSetCursorToEnd(containerRef, viewRef, true)
    } // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [autoFocusAndPosition, viewRef, containerRef, scrollToAndSetCursorToEnd, dirty])

  useEffect(() => {
    if (!isEmpty(templateData)) {
      const currentContent = viewRef.current?.state.doc.toString() ?? ''
      const newContent = decodeGitContent(templateData)

      // If empty dispatch template data, if content same as old content then dispatch previous content
      if (currentContent !== newContent) {
        viewRef.current?.dispatch({
          changes: {
            from: 0,
            to: currentContent.length, // You might need to adjust this based on how your content is structured
            insert: currentContent
          }
        })
      }
      if (isEmpty(currentContent)) {
        viewRef.current?.dispatch({
          changes: {
            from: 0,
            to: 0,
            insert: newContent
          }
        })
      }
    }
  }, [templateData])

  const setFileCallback = (newFile: File) => {
    setFile(newFile)
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handlePasteForSetFile = (event: { preventDefault: () => void; clipboardData: any }) => {
    handlePaste(event, setFileCallback)
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleDropForSetFile = async (event: any) => {
    handleFileDrop(event, setFileCallback)
  }

  useEffect(() => {
    const view = viewRef.current
    if (markdownContent && view) {
      const insertText = file?.type.startsWith('image/') ? `![image](${markdownContent})` : `${markdownContent}`
      view.dispatch(
        view.state.changeByRange(range => ({
          changes: [{ from: range.from, insert: insertText }],
          range: EditorSelection.range(range.from + insertText.length, range.from + insertText.length)
        }))
      )
    }
  }, [markdownContent]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleButtonClick = () => {
    if (fileInputRef.current) {
      fileInputRef.current.click()
    }
  }
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleFileChange = (event: any) => {
    setFile(event?.target?.files[0])
  }
  const onSaveHandler = useCallback(() => onSave?.(viewRef.current?.state.doc.toString() || ''), [onSave])

  return (
    <Container ref={containerRef} className={cx(css.container, { [css.noBorder]: noBorder }, className)}>
      <Dialog
        onClose={() => {
          setFile(undefined)
          setOpen(false)
        }}
        className={css.dialog}
        isOpen={open}>
        <Text font={{ variation: FontVariation.H4 }}>{getString('imageUpload.title')}</Text>

        <Container
          margin={{ top: 'small' }}
          onDragOver={event => {
            event.preventDefault()
          }}
          onDrop={handleDropForSetFile}
          onPaste={handlePasteForSetFile}
          flex={{ alignItems: 'center' }}
          className={css.uploadContainer}
          width={500}
          height={81}>
          {file ? (
            <Layout.Horizontal
              width={`100%`}
              padding={{ left: 'medium', right: 'medium' }}
              flex={{ justifyContent: 'space-between' }}>
              <Layout.Horizontal spacing="small">
                <Text lineClamp={1} width={200}>
                  {file.name}
                </Text>
                <Text>{formatBytes(file.size)}</Text>
              </Layout.Horizontal>
              <FlexExpander />
              <Text icon={'tick'} iconProps={{ color: Color.GREEN_800 }} color={Color.GREEN_800}>
                {getString('imageUpload.readyToUpload')}
              </Text>
            </Layout.Horizontal>
          ) : (
            <Text padding={{ left: 'medium' }} color={Color.GREY_400}>
              {getString('imageUpload.text')}
              <input type="file" ref={fileInputRef} onChange={handleFileChange} style={{ display: 'none' }} />
              <Button
                margin={{ left: 'small' }}
                text={getString('browse')}
                onClick={handleButtonClick}
                variation={ButtonVariation.SECONDARY}
              />
            </Text>
          )}
        </Container>
        <Container padding={{ top: 'large' }}>
          <Layout.Horizontal spacing="small">
            <Button
              type="submit"
              text={getString('imageUpload.upload')}
              variation={ButtonVariation.PRIMARY}
              disabled={false}
              onClick={() => {
                if (file !== undefined) {
                  handleUpload(file as File, setMarkdownContent, repoMetadata, showError, standalone, routingId)
                  setOpen(false)
                } else {
                  showError(getString('uploadAFileError'))
                }
              }}
            />
            <Button
              text={getString('cancel')}
              variation={ButtonVariation.TERTIARY}
              onClick={() => {
                setOpen(false)
                setFile(undefined)
              }}
            />
          </Layout.Horizontal>
        </Container>
      </Dialog>
      <ul className={css.tabs}>
        <li>
          <a
            role="tab"
            tabIndex={0}
            aria-selected={selectedTab === MarkdownEditorTab.WRITE}
            onClick={() => setSelectedTab(MarkdownEditorTab.WRITE)}>
            Write
          </a>
        </li>

        <li>
          <a
            role="tab"
            tabIndex={0}
            aria-selected={selectedTab === MarkdownEditorTab.PREVIEW}
            onClick={() => setSelectedTab(MarkdownEditorTab.PREVIEW)}>
            Preview
          </a>
        </li>
      </ul>
      <Container className={css.toolbar}>
        {outlets[CommentBoxOutletPosition.START_OF_MARKDOWN_EDITOR_TOOLBAR]}
        {toolbar.map((item, index) => {
          return (
            <Button
              key={index}
              size={ButtonSize.SMALL}
              variation={ButtonVariation.ICON}
              icon={item.icon}
              withoutCurrentColor
              iconProps={{ color: Color.PRIMARY_10, size: 14 }}
              onClick={() => onToolbarAction(item.action)}
            />
          )
        })}
      </Container>
      <Container className={css.tabContent}>
        <Editor
          extensions={[myKeymap, decorationField, history()]}
          routingId={routingId}
          standalone={standalone}
          repoMetadata={repoMetadata}
          forMarkdown
          content={value || ''}
          placeholder={i18n.placeHolder}
          autoFocus
          viewRef={viewRef}
          setDirty={setDirty}
          maxHeight={editorHeight}
          className={selectedTab === MarkdownEditorTab.PREVIEW ? css.hidden : undefined}
          onChange={(doc, _viewUpdate, _isDirty) => {
            if (_isDirty) {
              onChange?.(doc.toString())
            }
          }}
        />
        {selectedTab === MarkdownEditorTab.PREVIEW && (
          <MarkdownViewer source={viewRef.current?.state.doc.toString() || ''} maxHeight={800} />
        )}
      </Container>
      {!hideButtons && (
        <Container className={css.buttonsBar}>
          <Layout.Horizontal spacing="small">
            <Button disabled={!dirty} variation={ButtonVariation.PRIMARY} onClick={onSaveHandler} text={i18n.save} />
            {SecondarySaveButton && (
              <SecondarySaveButton disabled={!dirty} onClick={async () => await onSaveHandler()} />
            )}
            {!hideCancel && <Button variation={ButtonVariation.TERTIARY} onClick={onCancel} text={i18n.cancel} />}
          </Layout.Horizontal>
        </Container>
      )}
      {!isEmpty(data) && !generating && !standalone && (
        <AIDAFeedback
          className={css.aidaFeedback}
          allowCreateTicket={true}
          allowFeedback={true}
          telemetry={{
            aidaClient: AidaClient.CODE_PR_SUMMARY,
            metadata: {
              query: getString('generateSummary'),
              generatedResponse: isString(data) ? data : getString('invalidResponse')
            }
          }}
        />
      )}
    </Container>
  )
}

function scrollToAndSetCursorToEnd(
  containerRef: React.RefObject<HTMLDivElement>,
  viewRef: React.MutableRefObject<EditorView | undefined>,
  moveCursorToEnd = true
) {
  const dom = containerRef?.current as unknown as { scrollIntoViewIfNeeded: () => void }
  if (!dom) {
    return
  }
  // TODO: polyfill scrollintviewifneeded for other browsers besides chrome for scroll
  dom?.scrollIntoViewIfNeeded?.()

  if (moveCursorToEnd && viewRef.current) {
    const length = viewRef.current.state.doc.length
    viewRef.current.dispatch({ selection: { anchor: length, head: length } })
  }
}
