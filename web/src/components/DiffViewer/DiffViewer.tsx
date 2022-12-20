import React, { useCallback, useEffect, useRef, useState } from 'react'
import ReactDOM from 'react-dom'
import { useInView } from 'react-intersection-observer'
import { useResizeDetector } from 'react-resize-detector'
import { Button, Color, Container, FlexExpander, ButtonVariation, Layout, Text, ButtonSize } from '@harness/uicore'
import MarkdownEditor from '@uiw/react-markdown-editor'
import { indentWithTab } from '@codemirror/commands'
import cx from 'classnames'
import { keymap } from '@codemirror/view'
import { Diff2HtmlUI } from 'diff2html/lib-esm/ui/js/diff2html-ui'
import 'highlight.js/styles/github.css'
import 'diff2html/bundles/css/diff2html.min.css'
import { useStrings } from 'framework/strings'
import { CodeIcon } from 'utils/GitUtils'
import { useEventListener } from 'hooks/useEventListener'
import type { DiffFileEntry } from 'utils/types'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { DIFF2HTML_CONFIG, DIFF_VIEWER_HEADER_HEIGHT, INITIAL_ANNOTATION_HEIGHT, ViewStyle } from './DiffViewerUtils'
import css from './DiffViewer.module.scss'

interface AnnotationEntry {
  left: boolean
  right: boolean
  lineNumber: number
  width: number
  height: number
  rowNodeNthChildNumber: number
  element: HTMLDivElement | null
  contents: string[]
}

interface DiffViewerProps {
  diff: DiffFileEntry
  viewStyle: ViewStyle
  stickyTopPosition?: number
}

//
// Note: Lots of direct DOM manipulations are used to boost performance.
// Avoid React re-rendering at all cost as it might cause unresponsive UI
// when diff content is big, or when a PR has a lot of changed files.
//
export const DiffViewer: React.FC<DiffViewerProps> = ({ diff, viewStyle, stickyTopPosition = 0 }) => {
  const { getString } = useStrings()
  const [viewed, setViewed] = useState(false)
  const [collapsed, setCollapsed] = useState(false)
  const [fileUnchanged] = useState(diff.unchangedPercentage === 100)
  const [fileDeleted] = useState(diff.isDeleted)
  const [renderCustomContent, setRenderCustomContent] = useState(fileUnchanged || fileDeleted)
  const [height, setHeight] = useState<number | string>('auto')
  const [diffRenderer, setDiffRenderer] = useState<Diff2HtmlUI>()
  const { ref: inViewRef, inView } = useInView({ rootMargin: '100px 0px' })
  const containerRef = useRef<HTMLDivElement | null>(null)
  const [annotations, setAnnotations] = useState<AnnotationEntry[]>([])
  const setContainerRef = useCallback(
    node => {
      containerRef.current = node
      inViewRef(node)
    },
    [inViewRef]
  )
  const contentRef = useRef<HTMLDivElement>(null)
  const setupViewerInitialStates = useCallback(() => {
    setDiffRenderer(
      new Diff2HtmlUI(
        document.getElementById(diff.contentId) as HTMLElement,
        [diff],
        Object.assign({}, DIFF2HTML_CONFIG, { outputFormat: viewStyle })
      )
    )
  }, [diff, viewStyle])
  const renderDiffAndUpdateContainerHeightIfNeeded = useCallback(
    (enforced = false) => {
      const contentDOM = contentRef.current as HTMLDivElement
      const containerDOM = containerRef.current as HTMLDivElement

      if (!contentDOM.dataset.rendered || enforced) {
        if (!renderCustomContent || enforced) {
          containerDOM.style.height = 'auto'
          diffRenderer?.draw()
        }
        contentDOM.dataset.rendered = 'true'
        setHeight(containerDOM.clientHeight)
      }
    },
    [diffRenderer, renderCustomContent]
  )

  useEffect(
    function createDiffRenderer() {
      if (inView && !diffRenderer) {
        setupViewerInitialStates()
      }
    },
    [inView, diffRenderer, setupViewerInitialStates]
  )

  useEffect(
    function renderInitialContent() {
      if (diffRenderer) {
        const container = containerRef.current as HTMLDivElement
        const { classList: containerClassList } = container

        if (inView) {
          containerClassList.remove(css.offscreen)
          renderDiffAndUpdateContainerHeightIfNeeded()
        } else {
          containerClassList.add(css.offscreen)
        }
      }
    },
    [inView, diffRenderer, renderDiffAndUpdateContainerHeightIfNeeded]
  )

  useEffect(
    function handleCollapsedState() {
      const containerDOM = containerRef.current as HTMLDivElement & { scrollIntoViewIfNeeded: () => void }
      const { classList: containerClassList, style: containerStyle } = containerDOM

      if (collapsed) {
        containerClassList.add(css.collapsed)

        // Fix scrolling position messes up with sticky header
        const { y } = containerDOM.getBoundingClientRect()
        if (y - stickyTopPosition < 1) {
          containerDOM.scrollIntoView()

          if (stickyTopPosition) {
            window.scroll({ top: window.scrollY - stickyTopPosition })
          }
        }

        if (parseInt(containerStyle.height) != DIFF_VIEWER_HEADER_HEIGHT) {
          containerStyle.height = `${DIFF_VIEWER_HEADER_HEIGHT}px`
        }
      } else {
        containerClassList.remove(css.collapsed)

        const annotationHeights = annotations.reduce((total, annotation) => total + annotation.height, 0) || 0
        const newHeight = Number(height) + annotationHeights

        if (parseInt(containerStyle.height) != newHeight) {
          containerStyle.height = `${newHeight}px`
        }
      }
    },
    [collapsed, height, stickyTopPosition, annotations]
  )

  useEventListener(
    'click',
    useCallback(
      // TODO: Re-implement this function to just build data-structure in annotations
      // and not do any rendering. The rendering should be done by another function in
      // which it reads annotations and render them. By doing that then annotations can be
      // constructed from backend data and rendering process can work with both data
      // from backend and event from UI
      function clickToAnnotate(event: MouseEvent) {
        const isSideBySideView = viewStyle === ViewStyle.SIDE_BY_SIDE
        const target = event.target as HTMLDivElement
        const targetButton = target?.closest('[data-annotation-for-line]') as HTMLDivElement
        const annotatedLineRow = targetButton?.closest('tr') as HTMLTableRowElement

        const annotationEntry: AnnotationEntry = {
          left: false,
          right: false,
          lineNumber: 0,
          height: 0,
          width: 0,
          rowNodeNthChildNumber: 1,
          element: null,
          contents: []
        }

        if (targetButton && annotatedLineRow) {
          if (isSideBySideView) {
            const leftParent = targetButton.closest('.d2h-file-side-diff.left')
            annotationEntry.left = !!leftParent
            annotationEntry.right = !leftParent
            annotationEntry.lineNumber = Number(targetButton.dataset.annotationForLine)
          } else {
            const lineInfoTD = targetButton.closest('td')?.previousElementSibling
            const lineNum1 = lineInfoTD?.querySelector('.line-num1')
            const lineNum2 = lineInfoTD?.querySelector('.line-num2')

            annotationEntry.left = !!lineNum1?.textContent
            annotationEntry.right = !annotationEntry.left
            annotationEntry.lineNumber = Number(lineNum1?.textContent || lineNum2?.textContent)
          }

          annotatedLineRow.dataset.annotated = 'true'
          annotatedLineRow.dataset.line = String(annotationEntry.lineNumber)

          const annotationEntryRowElement = document.createElement('tr')
          annotationEntryRowElement.dataset.annotatedLine = String(annotationEntry.lineNumber)

          annotationEntry.height = INITIAL_ANNOTATION_HEIGHT

          annotationEntryRowElement.innerHTML = `
          <td colspan="2" class="${css.annotationCell}" height="${annotationEntry.height}px">
          <div class="${css.annotationContainer}" data-view-style="${viewStyle}"></div>
          </td>
          `
          annotatedLineRow.after(annotationEntryRowElement)
          annotationEntry.element = annotationEntryRowElement.querySelector(`.${css.annotationContainer}`)

          // TODO: Find a way to clean up ReactDOM.render() as it may leak memory
          // when we do inline like below

          // Render custom React inside element
          ReactDOM.unmountComponentAtNode(annotationEntry.element as HTMLDivElement)
          ReactDOM.render(<Comment />, annotationEntry.element)

          // Determine the location of the annotation inside its parent
          let node = annotatedLineRow as Element
          while (node.previousElementSibling) {
            annotationEntry.rowNodeNthChildNumber++
            node = node.previousElementSibling
          }

          // Split view: Calculate, inject, and adjust an empty place-holder row in the opposite pane
          if (isSideBySideView) {
            const filesDiff = annotatedLineRow.closest('.d2h-files-diff') as HTMLElement
            const sideDiff = filesDiff?.querySelector(`div.${annotationEntry.left ? 'right' : 'left'}`) as HTMLElement
            const sideRow = sideDiff?.querySelector(`tr:nth-child(${annotationEntry.rowNodeNthChildNumber})`)

            const tr2 = document.createElement('tr')
            tr2.innerHTML = `
            <td height="${annotationEntry.height}px" class="d2h-code-side-linenumber d2h-code-side-emptyplaceholder d2h-cntx d2h-emptyplaceholder"></td>
            <td class="d2h-cntx d2h-emptyplaceholder" height="${annotationEntry.height}px">
              <div class="d2h-code-side-line d2h-code-side-emptyplaceholder">
                <span class="d2h-code-line-prefix">&nbsp;</span>
                <span class="d2h-code-line-ctn hljs"><br></span>
              </div>
            </td>
            `
            sideRow?.after(tr2)
          }

          console.log(annotationEntry)

          setAnnotations([...annotations, annotationEntry])
        }
      },
      [viewStyle, annotations]
    ),
    containerRef.current as HTMLDivElement
  )

  return (
    <Container
      ref={setContainerRef}
      id={diff.containerId}
      className={css.main}
      style={{ '--diff-viewer-sticky-top': `${stickyTopPosition}px` } as React.CSSProperties}>
      <Layout.Vertical>
        <Container className={css.diffHeader} height={DIFF_VIEWER_HEADER_HEIGHT}>
          <Layout.Horizontal>
            <Button
              variation={ButtonVariation.ICON}
              icon={collapsed ? 'chevron-right' : 'chevron-down'}
              size={ButtonSize.SMALL}
              onClick={() => setCollapsed(!collapsed)}
            />
            <Container style={{ alignSelf: 'center' }} padding={{ right: 'small' }}>
              <Layout.Horizontal spacing="xsmall">
                {!!diff.addedLines && (
                  <Text color={Color.GREEN_600} style={{ fontSize: '12px' }}>
                    +{diff.addedLines}
                  </Text>
                )}
                {!!diff.addedLines && !!diff.deletedLines && <PipeSeparator height={8} />}
                {!!diff.deletedLines && (
                  <Text color={Color.RED_500} style={{ fontSize: '12px' }}>
                    -{diff.deletedLines}
                  </Text>
                )}
              </Layout.Horizontal>
            </Container>
            <Text inline className={css.fname}>
              {diff.isDeleted ? diff.oldName : diff.isRename ? `${diff.oldName} -> ${diff.newName}` : diff.newName}
            </Text>
            <Button
              variation={ButtonVariation.ICON}
              icon={CodeIcon.Copy}
              size={ButtonSize.SMALL}
              tooltip={
                <Container style={{ overflow: 'auto', width: 500, height: 400 }}>
                  <pre>{JSON.stringify(diff, null, 2)}</pre>
                </Container>
              }
            />
            <FlexExpander />
            <Container>
              <label className={css.viewLabel}>
                <input
                  type="checkbox"
                  value="viewed"
                  checked={viewed}
                  onChange={() => {
                    setViewed(!viewed)
                    setCollapsed(!viewed)
                  }}
                />
                {getString('viewed')}
              </label>
            </Container>
          </Layout.Horizontal>
        </Container>

        <Container id={diff.contentId} className={css.diffContent} ref={contentRef}>
          {renderCustomContent && (
            <Container>
              <Layout.Vertical
                padding="xlarge"
                style={{
                  alignItems: 'center'
                }}>
                {fileDeleted && (
                  <Button
                    variation={ButtonVariation.LINK}
                    onClick={() => {
                      setRenderCustomContent(false)
                      setTimeout(() => renderDiffAndUpdateContainerHeightIfNeeded(true), 0)
                    }}>
                    {getString('pr.showDiff')}
                  </Button>
                )}
                <Text>{getString(fileDeleted ? 'pr.fileDeleted' : 'pr.fileUnchanged')}</Text>
              </Layout.Vertical>
            </Container>
          )}
        </Container>
      </Layout.Vertical>
    </Container>
  )
}

const Comment = () => {
  const [contents, setContents] = useState<string[]>([])
  const [markdown, setMarkdown] = useState('')
  const resizeDetector = useResizeDetector()

  useEffect(() => {
    // console.log('resizeDetector.height', resizeDetector.height, 'annotationEntry.height', annotationEntry.height)
    // if (resizeDetector.height !== annotationEntry.height) {
    //   annotationEntry.height = resizeDetector.height as number
    //   setAnnotations([...annotations])
    // }
  }, [resizeDetector.height])

  return (
    <Layout.Vertical
      padding="xlarge"
      spacing="medium"
      className={cx(css.commentContainer, contents.length && css.hasContents)}
      ref={resizeDetector.ref}>
      {!!contents.length && (
        <Container>
          <Layout.Vertical spacing="large">
            {contents.map((content, index) => (
              <MarkdownEditor.Markdown key={index} source={content} />
            ))}
          </Layout.Vertical>
        </Container>
      )}
      <Container className={css.editorContainer}>
        <MarkdownEditor
          value={markdown}
          visible={false}
          placeholder={contents.length ? 'Reply here...' : 'Leave a comment here...'}
          theme="light"
          indentWithTab={false}
          autoFocus
          toolbars={[
            'header',
            'bold',
            'italic',
            // 'strike',
            'quote',
            'olist',
            'ulist',
            'todo',
            'link',
            'image',
            'codeBlock'
          ]}
          toolbarsMode={[]}
          basicSetup={{
            lineNumbers: false,
            foldGutter: false,
            highlightActiveLine: false
          }}
          extensions={[keymap.of([indentWithTab])]}
          onChange={(value, _viewUpdate) => setMarkdown(value)}
        />
      </Container>
      <Container className={css.actionsBar}>
        <Layout.Horizontal spacing="small">
          <Button
            disabled={!(markdown || '').trim()}
            variation={ButtonVariation.PRIMARY}
            onClick={() => {
              setContents([...contents, markdown])
              setMarkdown('')
            }}>
            {contents.length ? 'Comment' : 'Add comment'}
          </Button>
          {!contents.length && <Button variation={ButtonVariation.TERTIARY}>Cancel</Button>}
        </Layout.Horizontal>
      </Container>
    </Layout.Vertical>
  )
}
