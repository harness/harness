import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useInView } from 'react-intersection-observer'
import { Button, Color, Container, FlexExpander, ButtonVariation, Layout, Text, ButtonSize } from '@harness/uicore'
import type * as Diff2Html from 'diff2html'
import HoganJsUtils from 'diff2html/lib/hoganjs-utils'
import { Diff2HtmlUI } from 'diff2html/lib-esm/ui/js/diff2html-ui'
import 'highlight.js/styles/github.css'
import 'diff2html/bundles/css/diff2html.min.css'
import type { DiffFile } from 'diff2html/lib/types'
import { useStrings } from 'framework/strings'
import { CodeIcon } from 'utils/GitUtils'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import css from './DiffViewer.module.scss'

export enum ViewStyle {
  SPLIT = 'side-by-side',
  UNIFIED = 'line-by-line'
}

const DIFF_HEADER_HEIGHT = 36
const LINE_NUMBER_CLASS = 'diff-viewer-line-number'
const INITIAL_COMMENT_HEIGHT = 200

export const DIFF2HTML_CONFIG = {
  outputFormat: 'side-by-side',
  drawFileList: false,
  fileListStartVisible: false,
  fileContentToggle: true,
  matching: 'lines',
  synchronisedScroll: true,
  highlight: true,
  renderNothingWhenEmpty: false,
  compiledTemplates: {
    'generic-line': HoganJsUtils.compile(`
      <tr>
        <td class="{{lineClass}} {{type}} ${LINE_NUMBER_CLASS}" data-line-number="{{lineNumber}}">
          {{{lineNumber}}}
        </td>
        <td class="{{type}}" data-content-for-line-number="{{lineNumber}}">
            <div data-annotation-for-line="{{lineNumber}}" tab-index="0" role="button">+</div>
            <div class="{{contentClass}}">
            {{#prefix}}
                <span class="d2h-code-line-prefix">{{{prefix}}}</span>
            {{/prefix}}
            {{^prefix}}
                <span class="d2h-code-line-prefix">&nbsp;</span>
            {{/prefix}}
            {{#content}}
                <span class="d2h-code-line-ctn">{{{content}}}</span>
            {{/content}}
            {{^content}}
                <span class="d2h-code-line-ctn"><br></span>
            {{/content}}
            </div>
        </td>
      </tr>
    `),
    'side-by-side-file-diff': HoganJsUtils.compile(`
      <div id="{{fileHtmlId}}" class="d2h-file-wrapper side-by-side-file-diff" data-lang="{{file.language}}">
        <div class="d2h-file-header">
          {{{filePath}}}
        </div>
        <div class="d2h-files-diff">
            <div class="d2h-file-side-diff left">
                <div class="d2h-code-wrapper">
                    <table class="d2h-diff-table">
                        <tbody class="d2h-diff-tbody">
                        {{{diffs.left}}}
                        </tbody>
                    </table>
                </div>
            </div>
            <div class="d2h-file-side-diff right">
                <div class="d2h-code-wrapper">
                    <table class="d2h-diff-table">
                        <tbody class="d2h-diff-tbody">
                        {{{diffs.right}}}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
      </div>
    `),
    'line-by-line-file-diff': HoganJsUtils.compile(`
      <div id="{{fileHtmlId}}" class="d2h-file-wrapper line-by-line-file-diff" data-lang="{{file.language}}">
        <div class="d2h-file-header">
        {{{filePath}}}
        </div>
        <div class="d2h-file-diff">
            <div class="d2h-code-wrapper">
                <table class="d2h-diff-table">
                    <tbody class="d2h-diff-tbody">
                    {{{diffs}}}
                    </tbody>
                </table>
            </div>
        </div>
      </div>
    `),
    'line-by-line-numbers': HoganJsUtils.compile(`
      <div class="line-num1">{{oldNumber}}</div>
      <div class="line-num2">{{newNumber}}</div>
    `)
  }
} as Readonly<Diff2Html.Diff2HtmlConfig>

interface ContentAnnotationInfo {
  left: boolean
  right: boolean
  lineNumber: number
  annotatedElement: HTMLTableRowElement | null
  width: number
  height: number
  nthChild: number
}

interface DiffViewerProps {
  index: number
  diff: DiffFile
  viewStyle: ViewStyle
  stickyTopPosition?: number
}

//
// Note: Lots of direct DOM manipulations are used to boost performance
// Avoid React re-rendering at all cost as it might cause unresponsive UI
// when diff content is big, or when a PR has a lot of changed files.
//
export const DiffViewer: React.FC<DiffViewerProps> = ({ index, diff, viewStyle, stickyTopPosition = 0 }) => {
  const { getString } = useStrings()
  const [viewed, setViewed] = useState(false)
  const [collapsed, setCollapsed] = useState(false)
  const [fileUnchanged] = useState(diff.unchangedPercentage === 100)
  const [fileDeleted] = useState(diff.isDeleted)
  const [renderCustomContent, setRenderCustomContent] = useState(fileUnchanged || fileDeleted)
  const containerId = useMemo(() => `file-diff-container-${index}`, [index])
  const contentId = useMemo(() => `${containerId}-content`, [containerId])
  const [height, setHeight] = useState<number | string>('auto')
  const [diffRenderer, setDiffRenderer] = useState<Diff2HtmlUI>()
  const { ref: inViewRef, inView } = useInView({ rootMargin: '100px 0px' })
  const containerRef = useRef<HTMLDivElement | null>(null)
  const [annotations, setAnnotations] = useState<ContentAnnotationInfo[]>([])
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
        document.getElementById(contentId) as HTMLElement,
        [diff],
        Object.assign({}, DIFF2HTML_CONFIG, {
          outputFormat: viewStyle
        })
      )
    )
  }, [diff, contentId, viewStyle])
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

        if (parseInt(containerStyle.height) != DIFF_HEADER_HEIGHT) {
          containerStyle.height = `${DIFF_HEADER_HEIGHT}px`
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

  useEffect(
    function clickToAnnotate() {
      const onClick = (event: MouseEvent) => {
        const annotationInfo: ContentAnnotationInfo = {
          left: false,
          right: false,
          lineNumber: 0,
          annotatedElement: null,
          height: 0,
          width: 0,
          nthChild: 1
        }
        const target = event.target as HTMLDivElement
        const annotationButton = target?.closest('[data-annotation-for-line]') as HTMLDivElement
        const parentRow = annotationButton?.closest('tr') as HTMLTableRowElement
        const isSplitView = viewStyle === ViewStyle.SPLIT

        if (annotationButton && parentRow) {
          if (isSplitView) {
            const leftParent = annotationButton.closest('.d2h-file-side-diff.left')
            annotationInfo.left = !!leftParent
            annotationInfo.right = !leftParent
            annotationInfo.lineNumber = Number(annotationButton.dataset.lineNumber)
          } else {
            const lineInfoTD = annotationButton.closest('td')?.previousElementSibling
            const lineNum1 = lineInfoTD?.querySelector('.line-num1')
            const lineNum2 = lineInfoTD?.querySelector('.line-num2')

            annotationInfo.left = !!lineNum1?.textContent
            annotationInfo.right = !annotationInfo.left
            annotationInfo.lineNumber = Number(lineNum1?.textContent || lineNum2?.textContent)
          }

          const _height = INITIAL_COMMENT_HEIGHT

          parentRow.dataset.annotated = 'true' // TODO: set to lookup value instead of true
          annotationInfo.annotatedElement = parentRow
          annotationInfo.height = _height

          const tr = document.createElement('tr')
          tr.dataset.isAnnotation = 'true' // TODO: set to lookup value instead of true

          tr.innerHTML = `
          <td colspan="2" class="${css.annotationCell}" height="${annotationInfo.height}px">
            <div class="${css.annotationCellContainer}" data-view-style="${viewStyle}" contentEditable >Enter your comment or review here...</div>
          </td>
        `
          parentRow.after(tr)

          let node = parentRow as Element
          while (node.previousElementSibling) {
            annotationInfo.nthChild++
            node = node.previousElementSibling
          }

          // Add space in opposit pane (split view only)
          if (isSplitView) {
            const filesDiff = parentRow.closest('.d2h-files-diff') as HTMLElement
            const sideDiff = filesDiff?.querySelector(`div.${annotationInfo.left ? 'right' : 'left'}`) as HTMLElement
            const sideRow = sideDiff?.querySelector(`tr:nth-child(${annotationInfo.nthChild})`)

            const tr2 = document.createElement('tr')
            tr2.innerHTML = `<td colspan="2" class="${css.annotationCell}" height="${annotationInfo.height}px"></td>`

            sideRow?.after(tr2)
          }

          console.log(annotationInfo)

          setAnnotations([...annotations, annotationInfo])
        }
      }

      const containerDOM = containerRef.current as HTMLDivElement
      containerDOM.addEventListener('click', onClick)

      return () => {
        containerDOM.removeEventListener('click', onClick)
      }
    },
    [viewStyle, annotations]
  )

  return (
    <Container
      ref={setContainerRef}
      id={containerId}
      className={css.main}
      style={{ '--diff-viewer-sticky-top': `${stickyTopPosition}px` } as React.CSSProperties}>
      <Layout.Vertical>
        <Container className={css.diffHeader} height={DIFF_HEADER_HEIGHT}>
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
                Viewed
              </label>
            </Container>
          </Layout.Horizontal>
        </Container>

        <Container id={contentId} className={css.diffContent} ref={contentRef}>
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
