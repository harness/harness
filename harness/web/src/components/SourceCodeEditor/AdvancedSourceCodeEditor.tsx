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

import React, { forwardRef, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { debounce, get, isEmpty } from 'lodash-es'
import { parse } from 'yaml'
import { Range } from 'monaco-editor'
import type { EditorDidMount } from 'react-monaco-editor'
import { useStrings } from 'framework/strings'
import type { SourceCodeEditorProps } from 'utils/Utils'
import { CodeLensConfig, getDocumentSymbols, getPathFromRange, useCodeLenses } from 'hooks/useCodeLenses'
import type { EntityAddUpdateInterface } from 'components/PluginsPanel/PluginsPanel'
import { PipelineEntity, Action, CodeLensClickMetaData } from 'components/PipelineConfigPanel/types'
import { MonacoCodeEditorRef, SourceCodeEditorWithRef } from './SourceCodeEditorWithRef'
import { highlightInsertedYAML, setForwardedRef } from './EditorUtils'

import css from './AdvancedSourceCodeEditor.module.scss'

type AdvancedSourceCodeEditorProps = SourceCodeEditorProps & {
  enableCodeLens: boolean
  onEntityAddUpdate: (data: EntityAddUpdateInterface) => void
  onEntityFieldAddUpdate: (data: Partial<EntityAddUpdateInterface>) => void
}

const Editor = forwardRef<MonacoCodeEditorRef, AdvancedSourceCodeEditorProps>((props, ref) => {
  const { getString } = useStrings()
  const { onChange, onEntityAddUpdate, onEntityFieldAddUpdate } = props
  const editorRef = useRef<MonacoCodeEditorRef | null>(null)
  const [entityYAMLData, setEntityYAMLData] = useState<EntityAddUpdateInterface>()
  const [entityFieldData, setEntityFieldYAMLData] = useState<Partial<EntityAddUpdateInterface>>()
  const entityYAMLDataRef = useRef<EntityAddUpdateInterface>()
  const highlighterTimerIdRef = useRef<NodeJS.Timeout>()

  const editorDidMount: EditorDidMount = (editor, _monaco) => {
    editorRef.current = editor
    setForwardedRef(ref, editor)
  }

  useEffect(() => {
    if (!isEmpty(entityYAMLData)) {
      onEntityAddUpdate(entityYAMLData)
    }
  }, [entityYAMLData]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (entityFieldData) {
      onEntityFieldAddUpdate(entityFieldData)
    }
  }, [entityFieldData]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    return () => {
      if (highlighterTimerIdRef.current) {
        clearTimeout(highlighterTimerIdRef.current)
      }
    }
  }, [])

  const codeLensConfigs = useMemo<CodeLensConfig[]>(
    () => [
      {
        containerName: 'steps',
        commands: [
          {
            title: getString('edit'),
            onClick: (...args) => handleOnCodeLensClick(args),
            args: [
              {
                entity: PipelineEntity.STEP,
                action: Action.EDIT,
                highlightSelection: true
              } as CodeLensClickMetaData
            ]
          }
        ]
      },
      {
        name: 'steps',
        commands: [
          {
            title: getString('addLabel'),
            onClick: (...args) => handleOnCodeLensClick(args),
            args: [
              {
                entity: PipelineEntity.STEP,
                action: Action.ADD,
                highlightSelection: false
              } as CodeLensClickMetaData
            ]
          }
        ]
      }
    ],
    [] // eslint-disable-line react-hooks/exhaustive-deps
  )

  useCodeLenses({ editor: editorRef.current, codeLensConfigs })

  const handleOnCodeLensClick = useCallback(args => {
    try {
      const [{ path: pathToField, range }, metadata] = args
      const { entity: entityClicked, action: entityAction, highlightSelection } = metadata as CodeLensClickMetaData
      const model = editorRef.current?.getModel()
      if (pathToField && editorRef.current && model && model != null) {
        try {
          const entityData = {
            isUpdate: entityAction === Action.EDIT,
            pathToField,
            range,
            formData: get(parse(model.getValue() || ''), (pathToField as string[]).join('.')),
            ...{ entity: entityClicked, action: entityAction }
          }
          setEntityYAMLData(entityData)
          entityYAMLDataRef.current = entityData
        } catch (e) {
          // ignore parse error
        }
        if (highlightSelection)
          highlighterTimerIdRef.current = highlightInsertedYAML({
            range,
            editor: editorRef.current,
            style: css
          })
      }
    } catch (e) {
      // ignore error
    }
  }, [])

  /* Prepare info required to update a specific field inside an entity */
  const prepareEntityFieldUpdate = useCallback(async () => {
    try {
      const editor = editorRef.current
      if (editor && !isEmpty(entityYAMLDataRef.current)) {
        const { range: selectSymbolRange } = entityYAMLDataRef.current
        const currentPosition = editor.getPosition()
        /* Proceed with form UI update only if yaml update is relevant to the selected symbol */
        if (selectSymbolRange && currentPosition && Range.containsPosition(selectSymbolRange, currentPosition)) {
          const model = editor.getModel()
          if (model && model !== null) {
            const { lineNumber, column } = currentPosition || {}
            const symbols = await getDocumentSymbols(model)
            if (lineNumber && column) {
              const pathToField: string[] = getPathFromRange(
                { startLineNumber: lineNumber, startColumn: column, endLineNumber: lineNumber, endColumn: column },
                symbols
              )
              setEntityFieldYAMLData({
                pathToField,
                formData: get(parse(model.getValue() || ''), (pathToField as string[]).join('.'))
              })
            }
          }
        }
      }
    } catch (e) {
      // ignore error, including YAML parse error required on YAML change
    }
  }, [])

  const debouncedHandleYAMLUpdate = useMemo(
    () =>
      debounce((updatedYAML: string) => {
        onChange?.(updatedYAML)
        prepareEntityFieldUpdate()
      }, 300),
    [prepareEntityFieldUpdate]
  )

  return (
    <SourceCodeEditorWithRef
      {...props}
      onChange={debouncedHandleYAMLUpdate}
      ref={editorRef}
      editorOptions={{ codeLens: true }}
      editorDidMount={editorDidMount}
    />
  )
})

Editor.displayName = 'AdvancedSourceCodeEditor'

export const AdvancedSourceCodeEditor = React.memo(Editor)
