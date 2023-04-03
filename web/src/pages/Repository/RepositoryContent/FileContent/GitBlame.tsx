import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { throttle } from 'lodash-es'
import { Avatar, Container, FontVariation, Layout, StringSubstitute, Text } from '@harness/uicore'
import { LanguageDescription } from '@codemirror/language'
import { indentWithTab } from '@codemirror/commands'
import { ViewPlugin, ViewUpdate } from '@codemirror/view'
import { languages } from '@codemirror/language-data'
import { EditorView, gutter, GutterMarker, keymap, WidgetType } from '@codemirror/view'
import { Compartment, EditorState } from '@codemirror/state'
import ReactTimeago from 'react-timeago'
import { color } from '@uiw/codemirror-extensions-color'
import { hyperLink } from '@uiw/codemirror-extensions-hyper-link'
import { githubLight as theme } from '@uiw/codemirror-themes-all'
import { useGet } from 'restful-react'
import { Render } from 'react-jsx-match'
import type { GitBlameEntry, GitBlameResponse } from 'utils/types'
import type { GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { getErrorMessage } from 'utils/Utils'
import { lineWidget, LineWidgetPosition, LineWidgetSpec } from './lineWidget'
import css from './GitBlame.module.scss'

interface BlameBlock {
  fromLineNumber: number
  toLineNumber: number
  topPosition: number
  heights: Record<number, number>
  commitInfo: GitBlameEntry['Commit']
  lines: GitBlameEntry['Lines']
  numberOfLines: number
}

type BlameBlockRecord = Record<number, BlameBlock>

const BLAME_BLOCK_NOT_YET_CALCULATED_TOP_POSITION = -1

export const GitBlame: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'resourcePath'>> = ({
  repoMetadata,
  resourcePath
}) => {
  const { getString } = useStrings()
  const [blameBlocks, setBlameBlocks] = useState<BlameBlockRecord>({})
  const { data, error, loading } = useGet<GitBlameResponse>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/blame/${resourcePath}`,
    lazy: !repoMetadata || !resourcePath
  })

  useEffect(() => {
    if (data) {
      let fromLineNumber = 1

      data.forEach(({ Commit, Lines }) => {
        const toLineNumber = fromLineNumber + Lines.length - 1

        blameBlocks[fromLineNumber] = {
          fromLineNumber,
          toLineNumber,
          topPosition: BLAME_BLOCK_NOT_YET_CALCULATED_TOP_POSITION, // Not yet calculated
          heights: {}, // Not yet calculated
          commitInfo: Commit,
          lines: Lines,
          numberOfLines: Lines.length
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
    throttle(({ view, geometryChanged }: ViewUpdate) => {
      if (geometryChanged) {
        view.viewportLineBlocks.forEach(lineBlock => {
          const { from, top, height } = lineBlock
          const lineNumber = view.state.doc.lineAt(from).number
          const blameBlockAtLineNumber = findBlockForLineNumber(lineNumber)

          if (!blameBlockAtLineNumber) {
            // eslint-disable-next-line no-console
            console.error('Bad math! Cannot find a block at line', lineNumber)
          } else {
            if (blameBlockAtLineNumber.topPosition === BLAME_BLOCK_NOT_YET_CALCULATED_TOP_POSITION) {
              blameBlockAtLineNumber.topPosition = top
            }

            // CodeMirror reports top position of a block incorrectly sometimes, so we need to normalize it
            // using the previous block.
            if (lineNumber > 1) {
              const previousBlock = findBlockForLineNumber(lineNumber - 1)

              if (previousBlock.fromLineNumber !== blameBlockAtLineNumber.fromLineNumber) {
                const normalizedTop =
                  previousBlock.topPosition + Object.values(previousBlock.heights).reduce((a, b) => a + b, 0)

                blameBlockAtLineNumber.topPosition = normalizedTop
              }
            }

            blameBlockAtLineNumber.heights[lineNumber] = height
          }
        })

        setBlameBlocks({ ...blameBlocks })
      }
    }, 50),
    [blameBlocks]
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
          {Object.values(blameBlocks)
            .filter(({ topPosition }) => topPosition !== BLAME_BLOCK_NOT_YET_CALCULATED_TOP_POSITION)
            .map(({ fromLineNumber, topPosition: top, heights, commitInfo }) => {
              const height = Object.values(heights).reduce((a, b) => a + b, 0)

              return (
                <Container className={css.blameBox} key={fromLineNumber} height={height} style={{ top }}>
                  <Layout.Horizontal spacing="small" className={css.blameBoxLayout}>
                    <Container>
                      <Avatar name={commitInfo.Author.Identity.Name} size="normal" hoverCard={false} />
                    </Container>
                    <Container style={{ flexGrow: 1 }}>
                      <Layout.Vertical spacing="xsmall">
                        <Text
                          font={{ variation: FontVariation.BODY }}
                          lineClamp={2}
                          tooltipProps={{
                            portalClassName: css.blameCommitPortalClass
                          }}>
                          {commitInfo.Title}
                        </Text>
                        <Text font={{ variation: FontVariation.BODY }} lineClamp={1}>
                          <StringSubstitute
                            str={getString('blameCommitLine')}
                            vars={{
                              author: <strong>{commitInfo.Author.Identity.Name}</strong>,
                              timestamp: <ReactTimeago date={commitInfo.Author.When} />
                            }}
                          />
                        </Text>
                      </Layout.Vertical>
                    </Container>
                  </Layout.Horizontal>
                </Container>
              )
            })}
        </Container>
        <Render when={Object.values(blameBlocks).length}>
          <GitBlameSourceViewer
            source={data?.map(({ Lines }) => Lines.join('\n')).join('\n') || ''}
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

interface GitBlameSourceViewerProps {
  filename: string
  source: string
  onViewUpdate?: (update: ViewUpdate) => void
  blameBlocks: BlameBlockRecord
}

interface EditorLinePaddingWidgetSpec extends LineWidgetSpec {
  blockLines: number
}

function GitBlameSourceViewer({ source, filename, onViewUpdate, blameBlocks }: GitBlameSourceViewerProps) {
  const [view, setView] = useState<EditorView>()
  const ref = useRef<HTMLDivElement>()
  const languageConfig = useMemo(() => new Compartment(), [])
  const lineWidgetSpec = useMemo(() => {
    const spec: EditorLinePaddingWidgetSpec[] = []

    Object.values(blameBlocks).forEach(block => {
      const blockLines = block.numberOfLines

      spec.push({
        lineNumber: block.fromLineNumber,
        position: LineWidgetPosition.TOP,
        blockLines
      })

      spec.push({
        lineNumber: block.toLineNumber,
        position: LineWidgetPosition.BOTTOM,
        blockLines
      })
    })

    return spec
  }, [blameBlocks])

  useEffect(() => {
    const customLineNumberGutter = gutter({
      lineMarker(_view, line) {
        const lineNumber: number = _view.state.doc.lineAt(line.from).number
        return new CustomLineNumber(lineNumber)
      }
    })

    const editorView = new EditorView({
      doc: source,
      extensions: [
        customLineNumberGutter,

        ViewPlugin.fromClass(
          class {
            update(update: ViewUpdate) {
              onViewUpdate?.(update)
            }
          }
        ),

        color,
        hyperLink, // works pretty well in a markdown file
        theme,

        EditorView.lineWrapping,
        keymap.of([indentWithTab]),

        EditorState.readOnly.of(true),
        EditorView.editable.of(false),

        lineWidget({
          spec: lineWidgetSpec,
          widgetFor: spec => new EditorLinePaddingWidget(spec)
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

    setView(editorView)

    return () => {
      editorView.destroy()
    }
  }, []) // eslint-disable-line

  useEffect(() => {
    if (view && filename) {
      languageDescriptionFrom(filename)
        ?.load()
        .then(languageSupport => {
          view.dispatch({ effects: languageConfig.reconfigure(languageSupport) })
        })
    }
  }, [filename, view, languageConfig])

  return <Container ref={ref} className={css.main} />
}

function languageDescriptionFrom(filename: string) {
  return LanguageDescription.matchFilename(languages, filename)
}

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
