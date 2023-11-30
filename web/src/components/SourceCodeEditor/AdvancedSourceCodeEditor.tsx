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

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { debounce, get } from 'lodash-es'
import { parse } from 'yaml'
import { useStrings } from 'framework/strings'
import type { SourceCodeEditorProps } from 'utils/Utils'
import { CodeLensConfig, useCodeLenses } from 'hooks/useCodeLenses'
import type { EntityAddUpdateInterface } from 'components/PluginsPanel/PluginsPanel'
import { PipelineEntity, CodeLensAction, CodeLensClickMetaData } from 'components/PipelineConfigPanel/types'
import { MonacoCodeEditorRef, SourceCodeEditorWithRef } from './SourceCodeEditorWithRef'
import { highlightInsertedYAML } from './EditorUtils'

import css from './AdvancedSourceCodeEditor.module.scss'

type AdvancedSourceCodeEditorProps = SourceCodeEditorProps & {
  enableCodeLens: boolean
  onEntityAddUpdate: (data: EntityAddUpdateInterface) => void
}

function Editor(props: AdvancedSourceCodeEditorProps) {
  const { getString } = useStrings()
  const { onChange, onEntityAddUpdate } = props
  const editorRef = useRef<MonacoCodeEditorRef>(null)
  const [latestYAML, setLatestYAML] = useState<string>('')
  const [entityYAMLData, setEntityYAMLData] = useState<EntityAddUpdateInterface>()

  useEffect(() => {
    onChange?.(latestYAML)
  }, [latestYAML])

  useEffect(() => {
    if (entityYAMLData) {
      onEntityAddUpdate(entityYAMLData)
    }
  }, [entityYAMLData])

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
                action: CodeLensAction.EDIT,
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
                action: CodeLensAction.ADD,
                highlightSelection: false
              } as CodeLensClickMetaData
            ]
          }
        ]
      }
    ],
    []
  )

  useCodeLenses({ editor: editorRef.current, codeLensConfigs })

  const handleOnCodeLensClick = useCallback(
    args => {
      try {
        const [{ path: pathToField, range }, metadata] = args
        const { entity: entityClicked, action: entityAction, highlightSelection } = metadata as CodeLensClickMetaData
        if (pathToField && editorRef.current && editorRef.current.getModel() != null) {
          try {
            setEntityYAMLData({
              isUpdate: entityAction === CodeLensAction.EDIT,
              pathToField,
              formData: get(parse(editorRef.current.getModel()?.getValue() || ''), (pathToField as string[]).join('.')),
              ...{ entity: entityClicked, action: entityAction }
            })
          } catch (e) {
            // ignore parse error
          }
          if (highlightSelection)
            highlightInsertedYAML({
              range,
              editor: editorRef.current,
              style: css
            })
        }
      } catch (e) {
        // ignore error
      }
    },
    [editorRef.current]
  )

  const handleYAMLUpdate = useCallback((updatedYAML: string) => {
    setLatestYAML(updatedYAML)
  }, [])

  const debouncedHandleYAMLUpdate = useCallback(debounce(handleYAMLUpdate, 200), [handleYAMLUpdate])

  return (
    <SourceCodeEditorWithRef
      {...props}
      onChange={debouncedHandleYAMLUpdate}
      ref={editorRef}
      editorOptions={{ codeLens: true }}
    />
  )
}

export const AdvancedSourceCodeEditor = React.memo(Editor)
