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
import { Avatar, Container, Layout, StringSubstitute, Text } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import type { ViewUpdate } from '@codemirror/view'
import { EditorView, gutter, GutterMarker, WidgetType } from '@codemirror/view'
import { Compartment } from '@codemirror/state'
import ReactTimeago from 'react-timeago'
import { useGet } from 'restful-react'
import { Render } from 'react-jsx-match'
import { noop } from 'lodash-es'
import type { GitBlamePart, RepoRepositoryOutput } from 'services/code'
import { normalizeGitRef, type GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { getErrorMessage } from 'utils/Utils'
import { Editor } from 'components/Editor/Editor'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { useAppContext } from 'AppContext'
import { lineWidget, LineWidgetPosition, LineWidgetSpec } from './lineWidget'
import css from './GitBlame.module.scss'

interface BlameBlock {
  fromLineNumber: number
  toLineNumber: number
  topPosition: number
  heights: Record<number, number>
  commitInfo: GitBlamePart['commit']
  lines: GitBlamePart['lines']
  numberOfLines: number
}

interface BlameBlockExtended extends BlameBlock {
  repoMetaData?: RepoRepositoryOutput
}

type BlameBlockRecord = Record<number, BlameBlock>

const INITIAL_TOP_POSITION = -1

export const GitBlame: React.FC<
  Pick<GitInfoProps & { standalone: boolean }, 'repoMetadata' | 'resourcePath' | 'gitRef' | 'standalone'>
> = ({ repoMetadata, resourcePath, gitRef, standalone }) => {
  const { getString } = useStrings()
  const [blameBlocks, setBlameBlocks] = useState<BlameBlockRecord>({})
  const path = useMemo(
    () => `/api/v1/repos/${repoMetadata?.path}/+/blame/${resourcePath}`,
    [repoMetadata, resourcePath]
  )
  const {
    data: _data,
    error,
    loading
  } = useGet<GitBlamePart[]>({
    path,
    queryParams: {
      git_ref: normalizeGitRef(gitRef)
    },
    lazy: !repoMetadata || !resourcePath
  })
  const data = useMemo(() => {
    return (_data as unknown as string) === '[]' ? [] : _data
  }, [_data])

  useEffect(() => {
    if (data) {
      let fromLineNumber = 1

      data.forEach(({ commit, lines }) => {
        const toLineNumber = fromLineNumber + (lines?.length || 0) - 1

        blameBlocks[fromLineNumber] = {
          fromLineNumber,
          toLineNumber,
          topPosition: INITIAL_TOP_POSITION,
          heights: {},
          commitInfo: commit,
          lines: lines,
          numberOfLines: lines?.length || 0
        }

        fromLineNumber = toLineNumber + 1
      })

      setBlameBlocks({ ...blameBlocks })
    }
  }, [data]) // eslint-disable-line react-hooks/exhaustive-deps

  const findBlockForLineNumber = useCallback(
    lineNumber => {
      let startLine = lineNumber
      while (!blameBlocks[startLine] && startLine > 0) {
        startLine--
      }
      return blameBlocks[startLine]
    },
    [blameBlocks]
  )

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const onViewUpdate = useCallback(
    ({ view, geometryChanged }: ViewUpdate) => {
      if (geometryChanged) {
        view.viewportLineBlocks.forEach(lineBlock => {
          const { from, top, height } = lineBlock
          const lineNumber = view.state.doc.lineAt(from).number
          const blockAtLineNumber = findBlockForLineNumber(lineNumber)

          if (!blockAtLineNumber) {
            // eslint-disable-next-line no-console
            console.error('Bad math! Cannot find a blame block for line', lineNumber)
          } else {
            if (blockAtLineNumber.topPosition === INITIAL_TOP_POSITION) {
              blockAtLineNumber.topPosition = top
            }

            // CodeMirror reports top position of a block incorrectly sometimes, so we need to normalize it
            // using dimensions of the previous block.
            if (lineNumber > 1) {
              const previousBlock = findBlockForLineNumber(lineNumber - 1)

              if (previousBlock.fromLineNumber !== blockAtLineNumber.fromLineNumber) {
                blockAtLineNumber.topPosition = previousBlock.topPosition + computeHeight(previousBlock.heights)
              }
            }

            blockAtLineNumber.heights[lineNumber] = height

            const blockDOM = document.querySelector(
              `.${css.blameBox}[data-block-from-line="${blockAtLineNumber.fromLineNumber}"]`
            ) as HTMLDivElement

            if (blockDOM) {
              const _height = `${computeHeight(blockAtLineNumber.heights)}px`
              const _top = `${blockAtLineNumber.topPosition}px`

              if (blockDOM.style.height !== _height || blockDOM.style.top !== _top) {
                blockDOM.style.height = _height
                blockDOM.style.top = _top

                if (blockAtLineNumber.topPosition !== INITIAL_TOP_POSITION) {
                  blockDOM.removeAttribute('data-block-top')
                }
              }
            }
          }
        })
      }
    },
    [] // eslint-disable-line react-hooks/exhaustive-deps
  )

  // TODO: Normalize loading and error rendering when implementing new Design layout
  // that have Blame in a separate tab.
  if (loading) {
    return <Container padding={{ left: 'small' }}>{getString('loading')}</Container>
  }

  if (error) {
    return <Container padding="xlarge">{getErrorMessage(error)}</Container>
  }

  if (!data?.length) {
    return <Text font={{ variation: FontVariation.BODY }}>{getString('blameEmpty')}</Text>
  }

  return (
    <Container className={css.main}>
      <Layout.Horizontal className={css.layout}>
        <Container className={css.blameColumn}>
          {Object.values(blameBlocks).map(blameInfo => (
            <GitBlameMetaInfo repoMetaData={repoMetadata} key={blameInfo.fromLineNumber} {...blameInfo} />
          ))}
        </Container>

        <Render when={Object.values(blameBlocks).length}>
          <GitBlameRenderer
            standalone={standalone}
            repoMetadata={repoMetadata}
            source={data?.map(({ lines }) => (lines as string[]).join('\n')).join('\n') || ''}
            filename={resourcePath}
            onViewUpdate={onViewUpdate}
            blameBlocks={blameBlocks}
          />
        </Render>
      </Layout.Horizontal>
    </Container>
  )
}

class CustomLineNumber extends GutterMarker {
  lineNumber: number

  constructor(lineNumber: number) {
    super()
    this.lineNumber = lineNumber
  }

  toDOM() {
    const element = document.createElement('div')
    element.textContent = this.lineNumber.toString()
    element.classList.add(css.lineNo)
    return element
  }
}

interface GitBlameRendererProps {
  filename: string
  source: string
  onViewUpdate?: (update: ViewUpdate) => void
  blameBlocks: BlameBlockRecord
  repoMetadata: RepoRepositoryOutput | undefined
  standalone: boolean
}

interface EditorLinePaddingWidgetSpec extends LineWidgetSpec {
  blockLines: number
}

const GitBlameRenderer = React.memo(function GitBlameSourceViewer({
  source,
  filename,
  onViewUpdate = noop,
  blameBlocks,
  repoMetadata,
  standalone
}: GitBlameRendererProps) {
  const extensions = useMemo(() => new Compartment(), [])
  const viewRef = useRef<EditorView>()

  useEffect(() => {
    const customLineNumberGutter = gutter({
      lineMarker(_view, line) {
        const lineNumber: number = _view.state.doc.lineAt(line.from).number
        return new CustomLineNumber(lineNumber)
      }
    })
    const lineWidgetSpec: EditorLinePaddingWidgetSpec[] = []

    Object.values(blameBlocks).forEach(block => {
      const blockLines = block.numberOfLines

      lineWidgetSpec.push({
        lineNumber: block.fromLineNumber,
        position: LineWidgetPosition.TOP,
        blockLines
      })

      lineWidgetSpec.push({
        lineNumber: block.toLineNumber,
        position: LineWidgetPosition.BOTTOM,
        blockLines
      })
    })

    viewRef.current?.dispatch({
      effects: extensions.reconfigure([
        customLineNumberGutter,
        lineWidget({
          spec: lineWidgetSpec,
          widgetFor: spec => new EditorLinePaddingWidget(spec)
        })
      ])
    })
  }, [extensions, blameBlocks])

  return (
    <Editor
      standalone={standalone}
      inGitBlame={true}
      repoMetadata={repoMetadata}
      viewRef={viewRef}
      filename={filename}
      content={source}
      readonly={true}
      className={css.codeViewer}
      onViewUpdate={onViewUpdate}
      extensions={extensions.of([])}
      maxHeight="auto"
    />
  )
})

class EditorLinePaddingWidget extends WidgetType {
  constructor(readonly spec: EditorLinePaddingWidgetSpec) {
    super()
  }

  toDOM() {
    const { blockLines, position, lineNumber } = this.spec
    let height = 8

    if (position === LineWidgetPosition.BOTTOM && blockLines <= 4) {
      height += (5 - blockLines) * 15
    }

    const div = document.createElement('div')

    div.setAttribute('aria-hidden', 'true')
    div.setAttribute('data-line-number', String(lineNumber))
    div.setAttribute('data-position', position)
    div.style.height = `${height}px`

    return div
  }

  eq() {
    return false
  }

  ignoreEvent() {
    return false
  }
}

function computeHeight(heights: Record<number, number>) {
  return Object.values(heights).reduce((a, b) => a + b, 0)
}

function GitBlameMetaInfo({
  fromLineNumber,
  toLineNumber,
  topPosition,
  heights,
  commitInfo,
  repoMetaData
}: BlameBlockExtended) {
  const height = computeHeight(heights)
  const { getString } = useStrings()
  const { routes } = useAppContext()

  return (
    <Container
      className={css.blameBox}
      data-block-from-line={`${fromLineNumber}`}
      data-block-to-line={`${toLineNumber}`}
      data-block-top={`${topPosition}`}
      key={`${fromLineNumber}-${height}`}
      height={height}
      style={{ top: topPosition }}>
      <Layout.Horizontal spacing="small" className={css.blameBoxLayout}>
        <Container>
          <Avatar name={commitInfo?.author?.identity?.name} size="normal" hoverCard={false} />
        </Container>
        <Container style={{ flexGrow: 1 }}>
          <Layout.Vertical margin={{ right: 'xsmall' }} spacing="xsmall">
            <Text
              className={css.commitTitle}
              font={{ variation: FontVariation.BODY2_SEMI }}
              lineClamp={2}
              tooltipProps={{
                portalClassName: css.blameCommitPortalClass
              }}>
              {commitInfo?.title}
            </Text>
            <Text className={css.commitDate} font={{ variation: FontVariation.BODY }} lineClamp={1}>
              <StringSubstitute
                str={getString('blameCommitLine')}
                vars={{
                  author: <strong style={{ fontWeight: 500 }}>{commitInfo?.author?.identity?.name as string}</strong>,
                  timestamp: <ReactTimeago date={commitInfo?.author?.when as string} />
                }}
              />
            </Text>
          </Layout.Vertical>
        </Container>
        {commitInfo?.sha && repoMetaData?.path && (
          <Container style={{ float: 'right', position: 'relative' }}>
            <CommitActions
              sha={commitInfo?.sha}
              href={routes.toCODECommit({
                repoPath: repoMetaData.path,
                commitRef: commitInfo?.sha as string
              })}
              enableCopy
            />
          </Container>
        )}
      </Layout.Horizontal>
    </Container>
  )
}
