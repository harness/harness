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
import { Button, ButtonSize, ButtonVariation, Container, Layout, useToaster, Text } from '@harnessio/uicore'
import cx from 'classnames'
import { useParams } from 'react-router-dom'
import { useMutate } from 'restful-react'
import { Color, FontVariation } from '@harnessio/design-system'
import { PopoverPosition } from '@blueprintjs/core'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
import { useStrings } from 'framework/strings'
import type { OpenapiUpdatePullReqRequest, TypesListCommitResponse } from 'services/code'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import { NavigationCheck } from 'components/NavigationCheck/NavigationCheck'
import EnableAidaBanner from 'components/Aida/EnableAidaBanner'
import { CommentBoxOutletPosition, getErrorMessage } from 'utils/Utils'
import type { Identifier } from 'utils/types'
import Config from 'Config'
import { useAppContext } from 'AppContext'
import type { ConversationProps } from './Conversation'
import css from './Conversation.module.scss'

interface DescriptionBoxProps
  extends Omit<
    ConversationProps,
    'onCancelEditDescription' | 'pullReqCommits' | 'refetchActivities' | 'refetchPullReq'
  > {
  onCancelEditDescription: () => void
  pullReqCommits?: TypesListCommitResponse
}

export const DescriptionBox: React.FC<DescriptionBoxProps> = ({
  repoMetadata,
  pullReqMetadata,
  onDescriptionSaved,
  onCancelEditDescription,
  standalone,
  routingId
}) => {
  const { hooks } = useAppContext()

  const [flag, setFlag] = useState(false)
  const [edit, setEdit] = useState(false)
  const [dirty, setDirty] = useState(false)

  const [originalContent, setOriginalContent] = useState(pullReqMetadata.description as string)
  const [content, setContent] = useState(originalContent)
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { orgIdentifier, projectIdentifier } = useParams<Identifier>()
  const { data: aidaSettingResponse, loading: isAidaSettingLoading } = hooks?.useGetSettingValue({
    identifier: 'aida',
    queryParams: { accountIdentifier: routingId, orgIdentifier, projectIdentifier }
  })

  const { mutate } = useMutate({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata.number}`
  })

  useEffect(() => {
    setEdit(!pullReqMetadata?.description?.length)
    if (pullReqMetadata?.description) {
      setContent(pullReqMetadata?.description)
    }
  }, [pullReqMetadata?.description, pullReqMetadata?.description?.length])

  // write the above function handleCopilotClick in a callback
  const handleCopilotClick = useCallback(() => {
    setFlag(true)
  }, [])

  const handleDescUpdate = useCallback(
    (markdown: string) => {
      const payload: OpenapiUpdatePullReqRequest = {
        title: pullReqMetadata.title,
        description: markdown || ''
      }
      setOriginalContent(markdown)
      mutate(payload)
        .then(() => {
          setContent(markdown)
        })
        .catch(exception => showError(getErrorMessage(exception), 0, getString('pr.failedToUpdate')))
    },
    [getString, mutate, pullReqMetadata.title, showError]
  )

  const viewerDOMRef = useRef<HTMLElement>()

  useEffect(
    function toggleTodoCheck() {
      const dom = viewerDOMRef.current
      const TODO_LIST_MARKER = 'data-todo-index'
      const TODO_LIST_ITEM_CLASS = 'task-list-item'

      if (dom && !edit) {
        const handleClick = (e: MouseEvent) => {
          const targetIsListItem = (e.target as HTMLElement).classList.contains(TODO_LIST_ITEM_CLASS)
          const target = (e.target as HTMLElement)?.closest?.(`.${TODO_LIST_ITEM_CLASS}`)
          const input = target?.firstElementChild as HTMLInputElement
          const checked = targetIsListItem ? !input?.checked : input?.checked
          let sourceIndex = -1

          if (!input) return

          const index = Number(target?.getAttribute(TODO_LIST_MARKER))

          const newContent = originalContent
            .split('\n')
            .map(line => {
              if (line.startsWith('- [ ]') || line.startsWith('- [x]')) {
                sourceIndex++

                if (index === sourceIndex) {
                  return checked ? line.replace('- [ ]', '- [x]') : line.replace('- [x]', '- [ ]')
                }
              }
              return line
            })
            .join('\n')

          setContent(newContent)
          setOriginalContent(newContent)

          e.preventDefault()
          e.stopPropagation()

          handleDescUpdate(newContent)
        }

        // Enable all check inputs to allow clicking
        dom.querySelectorAll(`.${TODO_LIST_ITEM_CLASS} input`)?.forEach((input, index) => {
          input.removeAttribute('disabled')
          input.parentElement?.setAttribute(TODO_LIST_MARKER, String(index))
        })

        dom.addEventListener('click', handleClick)

        return () => dom.removeEventListener('click', handleClick)
      }
    },
    [edit, handleDescUpdate, originalContent]
  )

  return (
    <Container className={cx({ [css.box]: !edit, [css.desc]: !edit })}>
      <Container>
        {(edit && (
          <MarkdownEditorWithPreview
            routingId={routingId}
            standalone={standalone}
            repoMetadata={repoMetadata}
            value={content}
            flag={flag}
            targetGitRef={pullReqMetadata?.target_branch}
            sourceGitRef={pullReqMetadata?.source_branch}
            handleCopilotClick={handleCopilotClick}
            setFlag={setFlag}
            outlets={{
              [CommentBoxOutletPosition.START_OF_MARKDOWN_EDITOR_TOOLBAR]: (
                <>
                  {!isAidaSettingLoading && aidaSettingResponse?.data?.value == 'true' && !standalone ? (
                    <Button
                      size={ButtonSize.SMALL}
                      variation={ButtonVariation.ICON}
                      icon={'harness-copilot'}
                      withoutCurrentColor
                      iconProps={{
                        color: Color.GREY_0,
                        size: 22,
                        className: css.aidaIcon
                      }}
                      className={css.aidaIcon}
                      onClick={handleCopilotClick}
                      tooltip={
                        <Container padding={'small'} width={270}>
                          <Layout.Vertical flex={{ align: 'center-center' }}>
                            <Text font={{ variation: FontVariation.BODY }}>{getString('prGenSummary')}</Text>
                          </Layout.Vertical>
                        </Container>
                      }
                      tooltipProps={{
                        interactionKind: 'hover',
                        usePortal: true,
                        position: PopoverPosition.BOTTOM_LEFT,
                        popoverClassName: cx(css.popoverDescriptionbox)
                      }}
                    />
                  ) : null}
                </>
              ),
              [CommentBoxOutletPosition.ENABLE_AIDA_PR_DESC_BANNER]: <EnableAidaBanner />
            }}
            onSave={value => {
              if (value?.split('\n').some(line => line.length > Config.MAX_TEXT_LINE_SIZE_LIMIT)) {
                return showError(getString('pr.descHasTooLongLine', { max: Config.MAX_TEXT_LINE_SIZE_LIMIT }), 0)
              }

              if (value.length > Config.PULL_REQUEST_DESCRIPTION_SIZE_LIMIT) {
                return showError(
                  getString('pr.descIsTooLong', { max: Config.PULL_REQUEST_DESCRIPTION_SIZE_LIMIT, len: value.length }),
                  0
                )
              }

              const payload: OpenapiUpdatePullReqRequest = {
                title: pullReqMetadata.title,
                description: value || ''
              }
              mutate(payload)
                .then(() => {
                  setContent(value)
                  setOriginalContent(value)
                  setEdit(false)
                  onDescriptionSaved()
                })
                .catch(exception => showError(getErrorMessage(exception), 0, getString('pr.failedToUpdate')))
            }}
            onCancel={() => {
              setContent(originalContent)
              setEdit(false)
              onCancelEditDescription()
            }}
            setDirty={setDirty}
            i18n={{
              placeHolder: getString('pr.enterDesc'),
              tabEdit: getString('write'),
              tabPreview: getString('preview'),
              save: getString('save'),
              cancel: getString('cancel')
            }}
            editorHeight="400px"
            autoFocusAndPosition={true}
          />
        )) || (
          <React.Fragment key={originalContent}>
            <Container className={css.mdWrapper} ref={viewerDOMRef}>
              <MarkdownViewer source={content} />
              <Container className={css.menuWrapper}>
                <OptionsMenuButton
                  isDark={true}
                  icon="Options"
                  iconProps={{ size: 14 }}
                  style={{ padding: '5px' }}
                  items={[
                    {
                      text: getString('edit'),
                      className: css.optionMenuIcon,
                      hasIcon: true,
                      iconName: 'Edit',
                      onClick: () => setEdit(true)
                    }
                  ]}
                />
              </Container>
            </Container>
          </React.Fragment>
        )}
      </Container>
      <NavigationCheck when={dirty} />
    </Container>
  )
}
