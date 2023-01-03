import React, { useCallback, useEffect, useRef, useState } from 'react'
import ReactDOM from 'react-dom'
import { useInView } from 'react-intersection-observer'
import {
  Button,
  Color,
  Container,
  FlexExpander,
  ButtonVariation,
  Layout,
  Text,
  ButtonSize,
  Intent
} from '@harness/uicore'
import { Diff2HtmlUI } from 'diff2html/lib-esm/ui/js/diff2html-ui'
import 'highlight.js/styles/github.css'
import 'diff2html/bundles/css/diff2html.min.css'
import { noop } from 'lodash-es'
import { useStrings } from 'framework/strings'
import { CodeIcon } from 'utils/GitUtils'
import { useEventListener } from 'hooks/useEventListener'
import type { DiffFileEntry } from 'utils/types'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { useConfirmAction } from 'hooks/useConfirmAction'
import { useAppContext } from 'AppContext'
import {
  DIFF2HTML_CONFIG,
  DiffCommentItem,
  DIFF_VIEWER_HEADER_HEIGHT,
  getCommentLineInfo,
  renderCommentOppositePlaceHolder,
  ViewStyle
} from './DiffViewerUtils'
import { CommentBox } from '../CommentBox/CommentBox'
import css from './DiffViewer.module.scss'

interface DiffViewerProps {
  index: number
  diff: DiffFileEntry
  viewStyle: ViewStyle
  stickyTopPosition?: number
}

//
// Note: Lots of direct DOM manipulations are used to boost performance.
//       Avoid React re-rendering at all cost as it might cause unresponsive UI
//       when diff content is big, or when a PR has a lot of changed files.
//
export const DiffViewer: React.FC<DiffViewerProps> = ({ diff, index, viewStyle, stickyTopPosition = 0 }) => {
  const { getString } = useStrings()
  const [viewed, setViewed] = useState(false)
  const [collapsed, setCollapsed] = useState(false)
  const [fileUnchanged] = useState(diff.unchangedPercentage === 100)
  const [fileDeleted] = useState(diff.isDeleted)
  const [renderCustomContent, setRenderCustomContent] = useState(fileUnchanged || fileDeleted)
  const [heightWithoutComments, setHeightWithoutComents] = useState<number | string>('auto')
  const [diffRenderer, setDiffRenderer] = useState<Diff2HtmlUI>()
  const { ref: inViewRef, inView } = useInView({ rootMargin: '100px 0px' })
  const containerRef = useRef<HTMLDivElement | null>(null)
  const { currentUser } = useAppContext()
  const executeDeleteComentConfirmation = useConfirmAction({
    title: getString('delete'),
    intent: Intent.DANGER,
    message: <Text>{getString('deleteCommentConfirm')}</Text>,
    action: async ({ commentEntry, onSuccess = noop }) => {
      // TODO: Delete comment
      onSuccess('Delete ', commentEntry)
    }
  })

  const [comments, setComments] = useState<DiffCommentItem[]>(
    !index
      ? [
          {
            left: false,
            right: true,
            height: 0,
            lineNumber: 11,
            commentItems: [
              `Logs will looks similar to\n\n<img width="1494" alt="image" src="https://user-images.githubusercontent.com/98799615/207994246-19ce9eb2-604f-4226-9a3c-6f4125d3b7cc.png">\n\n\ngitrpc logs using the \`ctx\` will have the following annotations:\n- \`grpc.service=rpc.ReferenceService\`\n- \`grpc.method=CreateBranch\`\n- \`grpc.peer=127.0.0.1:49364\`\n- \`grpc.request_id=cedrl6p1eqltblt13mgg\``,
              `it seems we don't actually do anything with the explicit error type other than calling .Error(), which technically we could do on the original err object too? unless I'm missing something, could we then use errors.Is instead? (would avoid the extra var definitions at the top)`
              //`If error is not converted then it will be detailed error: in BranchDelete: Branch doesn't exists. What we want is human readable error: Branch 'name' doesn't exists.`,
              //  `* GitRPC isolated errors, bcoz this will be probably separate repo in future and we dont want every where to include grpc status codes in our main app\n* Errors are explicit for repsonses based on error passing by types`,
              // `> global ctx in wire will kill all routines, right? is this affect middlewares and interceptors? because requests should finish they work, right?\n\nI've changed the code now to pass the config directly instead of the systemstore and context, to avoid confusion (what we discussed yesterday - I remove systemstore itself another time).`
            ].map(content => ({
              author: 'Tan Nhu',
              created: '2022-12-21',
              updated: '2022-12-21',
              content
            }))
          }
        ]
      : []
  )
  const commentsRef = useRef<DiffCommentItem[]>(comments)
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
        setHeightWithoutComents(containerDOM.clientHeight)
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
      if (diffRenderer && inView) {
        renderDiffAndUpdateContainerHeightIfNeeded()
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

        // Fix scrolling position messes up with sticky header: When content of the diff content
        // is above the diff header, we need to scroll it back to below the header, adjust window
        // scrolling position to avoid the next diff scroll jump
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

        const commentsHeight = comments.reduce((total, comment) => total + comment.height, 0) || 0
        const newHeight = Number(heightWithoutComments) + commentsHeight

        if (parseInt(containerStyle.height) != newHeight) {
          containerStyle.height = `${newHeight}px`
        }
      }
    },
    [collapsed, heightWithoutComments, stickyTopPosition, comments]
  )

  useEventListener(
    'click',
    useCallback(
      function clickToAddAnnotation(event: MouseEvent) {
        const target = event.target as HTMLDivElement
        const targetButton = target?.closest('[data-annotation-for-line]') as HTMLDivElement
        const annotatedLineRow = targetButton?.closest('tr') as HTMLTableRowElement

        const commentItem: DiffCommentItem = {
          left: false,
          right: false,
          lineNumber: 0,
          height: 0,
          commentItems: []
        }

        if (targetButton && annotatedLineRow) {
          if (viewStyle === ViewStyle.SIDE_BY_SIDE) {
            const leftParent = targetButton.closest('.d2h-file-side-diff.left')
            commentItem.left = !!leftParent
            commentItem.right = !leftParent
            commentItem.lineNumber = Number(targetButton.dataset.annotationForLine)
          } else {
            const lineInfoTD = targetButton.closest('td')?.previousElementSibling
            const lineNum1 = lineInfoTD?.querySelector('.line-num1')
            const lineNum2 = lineInfoTD?.querySelector('.line-num2')

            // Right has priority
            commentItem.right = !!lineNum2?.textContent
            commentItem.left = !commentItem.right
            commentItem.lineNumber = Number(lineNum2?.textContent || lineNum1?.textContent)
          }

          setComments([...comments, commentItem])
        }
      },
      [viewStyle, comments]
    ),
    containerRef.current as HTMLDivElement
  )

  useEffect(
    function renderAnnotatations() {
      const isSideBySide = viewStyle === ViewStyle.SIDE_BY_SIDE

      // Update latest commentsRef to use it inside CommentBox callbacks
      commentsRef.current = comments

      comments.forEach(comment => {
        const lineInfo = getCommentLineInfo(contentRef.current, comment, viewStyle)

        if (lineInfo.rowElement) {
          const { rowElement } = lineInfo

          if (lineInfo.hasCommentsRendered) {
            if (isSideBySide) {
              const filesDiff = rowElement?.closest('.d2h-files-diff') as HTMLElement
              const sideDiff = filesDiff?.querySelector(`div.${comment.left ? 'right' : 'left'}`) as HTMLElement
              const oppositeRowPlaceHolder = sideDiff?.querySelector(
                `tr[data-place-holder-for-line="${comment.lineNumber}"]`
              )

              const first = oppositeRowPlaceHolder?.firstElementChild as HTMLTableCellElement
              const last = oppositeRowPlaceHolder?.lastElementChild as HTMLTableCellElement

              if (first && last) {
                first.style.height = `${comment.height}px`
                last.style.height = `${comment.height}px`
              }
            }
          } else {
            // Mark row that it has comment/annotation
            rowElement.dataset.annotated = 'true'

            // Create a new row below it and render CommentBox inside
            const commentRowElement = document.createElement('tr')
            commentRowElement.dataset.annotatedLine = String(comment.lineNumber)
            commentRowElement.innerHTML = `<td colspan="2"></td>`
            rowElement.after(commentRowElement)

            const element = commentRowElement.firstElementChild as HTMLTableCellElement

            // Note: CommentBox is rendered as an independent React component
            //       everything passed to it must be either values, or refs. If you
            //       pass callbacks or states, they won't be updated and might
            // .     cause unexpected bugs
            ReactDOM.unmountComponentAtNode(element as HTMLDivElement)
            ReactDOM.render(
              <CommentBox
                commentItems={comment.commentItems}
                getString={getString}
                width={isSideBySide ? 'calc(100vw / 2 - 163px)' : undefined}
                onHeightChange={boxHeight => {
                  if (comment.height !== boxHeight) {
                    comment.height = boxHeight
                    // element.style.height = `${boxHeight}px`
                    setTimeout(() => setComments([...commentsRef.current]), 0)
                  }
                }}
                onCancel={() => {
                  // Clean up CommentBox rendering and reset states bound to lineInfo
                  ReactDOM.unmountComponentAtNode(element as HTMLDivElement)
                  commentRowElement.parentElement?.removeChild(commentRowElement)
                  lineInfo.oppositeRowElement?.parentElement?.removeChild(
                    lineInfo.oppositeRowElement?.nextElementSibling as Element
                  )
                  delete lineInfo.rowElement.dataset.annotated
                  setTimeout(() => setComments(commentsRef.current.filter(item => item !== comment)), 0)
                }}
                currentUserName={currentUser.display_name}
                executeDeleteComent={executeDeleteComentConfirmation}
              />,
              element
            )

            // Split view: Calculate, inject, and adjust an empty place-holder row in the opposite pane
            if (isSideBySide && lineInfo.oppositeRowElement) {
              renderCommentOppositePlaceHolder(comment, lineInfo.oppositeRowElement)
            }
          }
        }
        // Comment no longer has UI relevant anchors to be rendered
        // else {
        //   console.info('Comment is discarded due to no UI relevant anchors', { comment, lineInfo })
        // }
      })
    },
    [comments, viewStyle, getString, currentUser, executeDeleteComentConfirmation]
  )

  useEffect(function cleanUpCommentBoxRendering() {
    const contentDOM = contentRef.current
    return () => {
      contentDOM
        ?.querySelectorAll('[data-annotated-line]')
        .forEach(element => ReactDOM.unmountComponentAtNode(element.firstElementChild as HTMLTableCellElement))
    }
  }, [])

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
            <Button variation={ButtonVariation.ICON} icon={CodeIcon.Copy} size={ButtonSize.SMALL} />
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
