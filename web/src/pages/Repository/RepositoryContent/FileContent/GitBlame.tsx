import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Avatar, Container, FontVariation, Layout, StringSubstitute, Text } from '@harness/uicore'
import type { ViewUpdate } from '@codemirror/view'
import { EditorView, gutter, GutterMarker, WidgetType } from '@codemirror/view'
import { Compartment } from '@codemirror/state'
import ReactTimeago from 'react-timeago'
import { useGet } from 'restful-react'
import { Render } from 'react-jsx-match'
import { noop } from 'lodash-es'
import type { GitrpcBlamePart } from 'services/code'
import type { GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { getErrorMessage } from 'utils/Utils'
import { Editor } from 'components/Editor/Editor'
import { lineWidget, LineWidgetPosition, LineWidgetSpec } from './lineWidget'
import css from './GitBlame.module.scss'

interface BlameBlock {
  fromLineNumber: number
  toLineNumber: number
  topPosition: number
  heights: Record<number, number>
  commitInfo: GitrpcBlamePart['commit']
  lines: GitrpcBlamePart['lines']
  numberOfLines: number
}

type BlameBlockRecord = Record<number, BlameBlock>

const INITIAL_TOP_POSITION = -1

export const GitBlame: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'resourcePath'>> = ({
  repoMetadata,
  resourcePath
}) => {
  const { getString } = useStrings()
  const [blameBlocks, setBlameBlocks] = useState<BlameBlockRecord>({})
  const { data, error, loading } = useGet<GitrpcBlamePart[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/blame/${resourcePath}`,
    lazy: !repoMetadata || !resourcePath
  })

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
    return <Container padding="xlarge">{getString('loading')}</Container>
  }

  if (error) {
    return <Container padding="xlarge">{getErrorMessage(error)}</Container>
  }

  return (
    <Container className={css.gitBlame}>
      <Layout.Horizontal className={css.layout}>
        <Container className={css.blameColumn}>
          {Object.values(blameBlocks).map(blameInfo => (
            <GitBlameMetaInfo key={blameInfo.fromLineNumber} {...blameInfo} />
          ))}
        </Container>
        <Render when={Object.values(blameBlocks).length}>
          <GitBlameRenderer
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
}

interface EditorLinePaddingWidgetSpec extends LineWidgetSpec {
  blockLines: number
}

const GitBlameRenderer = React.memo(function GitBlameSourceViewer({
  source,
  filename,
  onViewUpdate = noop,
  blameBlocks
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
      viewRef={viewRef}
      filename={filename}
      content={source}
      readonly={true}
      className={css.main}
      onViewUpdate={onViewUpdate}
      extensions={extensions.of([])}
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

function GitBlameMetaInfo({ fromLineNumber, toLineNumber, topPosition, heights, commitInfo }: BlameBlock) {
  const height = computeHeight(heights)
  const { getString } = useStrings()

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
          <Layout.Vertical spacing="xsmall">
            <Text
              font={{ variation: FontVariation.BODY }}
              lineClamp={2}
              tooltipProps={{
                portalClassName: css.blameCommitPortalClass
              }}>
              {commitInfo?.title}
            </Text>
            <Text font={{ variation: FontVariation.BODY }} lineClamp={1}>
              <StringSubstitute
                str={getString('blameCommitLine')}
                vars={{
                  author: <strong>{commitInfo?.author?.identity?.name as string}</strong>,
                  timestamp: <ReactTimeago date={commitInfo?.author?.when as string} />
                }}
              />
            </Text>
          </Layout.Vertical>
        </Container>
      </Layout.Horizontal>
    </Container>
  )
}
