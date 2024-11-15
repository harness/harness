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

import { Button, ButtonVariation, Container, Layout, Text, Utils, stringSubstitute } from '@harnessio/uicore'
import React, { useEffect, useMemo, useRef, useState } from 'react'
import { useAtom } from 'jotai'
import { refractor } from 'refractor'
import { Else, Match, Truthy } from 'react-jsx-match'
import { toHtml } from 'hast-util-to-html'
import type { Nodes } from 'hast-util-to-html/lib'
import { useStrings } from 'framework/strings'
import type { SuggestionBlock } from 'components/SuggestionBlock/SuggestionBlock'
import { Suggestion, pullReqSuggestionsAtom } from 'atoms/pullReqSuggestions'
import {
  useCommitPullReqSuggestions,
  useCommitSuggestionsModal
} from 'components/CommitModalButton/useCommitSuggestionModal'
import { PullRequestSection, getErrorMessage, waitUntil } from 'utils/Utils'
import { PullReqCustomEvent, getActivePullReqPageSection } from 'pages/PullRequest/PullRequestUtils'
import { dispatchCustomEvent } from 'hooks/useEventListener'
import css from './MarkdownViewer.module.scss'

interface CodeSuggestionBlockProps {
  code: string
  suggestionBlock?: SuggestionBlock
  suggestionCheckSums?: string[]
}
//
// NOTE: Adding this component to MarkdownViewer is not ideal as
// it makes MarkdownViewer less independent. It'd be better to adopt
// concept such as Outlet to make the Code Suggestion business on its own
//
export const CodeSuggestionBlock: React.FC<CodeSuggestionBlockProps> = ({
  code,
  suggestionBlock,
  suggestionCheckSums
}) => {
  const { getString } = useStrings()
  const codeBlockContent = suggestionBlock?.source || ''
  const lang = suggestionBlock?.lang || 'plaintext'
  const language = `language-${lang}`
  const html1 = toHtml(refractor.highlight(codeBlockContent, lang) as unknown as Nodes)
  const html2 = toHtml(refractor.highlight(code, lang) as unknown as Nodes)
  const ref = useRef<HTMLDivElement>(null)
  const suggestionRef = useRef<Suggestion>()
  const [checksum, setChecksum] = useState('')

  // TODO: Use `fast-diff` to decorate `removed, `added` blocks
  // Similar to Github. Otherwise, it looks plain
  // https://codesandbox.io/p/sandbox/intelligent-noether-3qd6mj?file=%2Fsrc%2FApp.js%3A1%2C19-1%2C28
  // Flow:
  // For removed block: Scan fast diff result, if a removed block is matched, mark bg red
  // For added block: Scan fast diff result, if an added block is matched, mark bg green

  // Notes: Since the suggestion checksums are on the comment level (JSON), and the suggestions themselves are
  // embedded in the comment content (Text), which make them be nothing related in terms of structure. We need
  // a way to link them together:
  // 1- Render suggestion block, each being marked with the comment
  // 2- When rendering is complete, we query all suggestions block and match each block to its check sum
  //    by index.
  useEffect(() => {
    const commentId = suggestionBlock?.commentId

    if (commentId && suggestionCheckSums?.length && ref.current) {
      const parent = ref.current.closest(`[data-comment-id="${commentId}"]`)
      const suggestionBlockDOMs = parent?.querySelectorAll(`[data-suggestion-comment-id="${commentId}"]`)
      let index = 0

      if (suggestionBlockDOMs?.length) {
        while (suggestionBlockDOMs[index]) {
          if (suggestionBlockDOMs[index] === ref.current) {
            setChecksum(suggestionCheckSums[index])
            break
          }
          index++
        }
      }
    }
  }, [code, suggestionBlock?.commentId, suggestionCheckSums])

  const text = useMemo(() => stringSubstitute(getString('pr.commitSuggestions'), { count: 1 }), [getString])
  const [suggestions, setSuggestions] = useAtom(pullReqSuggestionsAtom)
  const commitPullReqSuggestions = useCommitPullReqSuggestions()
  const [openCommitSuggestionsModal] = useCommitSuggestionsModal({
    title: text as string,
    commitMessage: stringSubstitute(getString('pr.applySuggestions'), { count: 1 }) as string,
    onCommit: async formData => {
      return new Promise(resolve => {
        commitPullReqSuggestions({
          bypass_rules: true,
          dry_run_rules: false,
          title: formData.commitMessage,
          message: formData.extendedDescription,
          suggestions: [suggestionRef.current]
        })
          .then(() => {
            resolve(null)

            switch (getActivePullReqPageSection()) {
              case PullRequestSection.FILES_CHANGED:
                waitUntil({
                  test: () => document.querySelector('[data-button-name="refresh-pr"]') as HTMLElement,
                  onMatched: dom => {
                    dom?.click?.()
                  },
                  onExpired: () => {
                    dispatchCustomEvent(PullReqCustomEvent.REFETCH_DIFF, null)
                  }
                })
                break

              case PullRequestSection.CONVERSATION:
                // Activities are refetched by SSE event, nothing to do here
                break
            }
          })
          .catch(e => {
            resolve(getErrorMessage(e))
          })
      })
    }
  })

  useEffect(() => {
    suggestionRef.current = {
      check_sum: checksum,
      comment_id: suggestionBlock?.commentId as number
    }
  }, [checksum, suggestionBlock?.commentId])

  const states = useMemo(
    () => ({
      addedToBatch: suggestions?.find(
        item => item.check_sum === checksum && item.comment_id === suggestionBlock?.commentId
      ),
      otherAddedToBatch: suggestions?.find(
        item => item.check_sum !== checksum && item.comment_id === suggestionBlock?.commentId
      )
    }),
    [suggestions, checksum, suggestionBlock]
  )

  const actions = useMemo(
    () => ({
      addToBatch: () => {
        setSuggestions([...suggestions, { check_sum: checksum, comment_id: suggestionBlock?.commentId as number }])
      },
      removeFromBatch: () => {
        setSuggestions(
          suggestions.filter(
            suggestion => !(suggestion.check_sum === checksum && suggestion.comment_id === suggestionBlock?.commentId)
          )
        )
      },
      commit: openCommitSuggestionsModal
    }),
    [checksum, suggestionBlock, suggestions, setSuggestions, openCommitSuggestionsModal]
  )

  return (
    <Container
      ref={ref}
      className={css.suggestion}
      onClick={Utils.stopEvent}
      data-suggestion-comment-id={suggestionBlock?.commentId}>
      <Layout.Vertical>
        <Text className={css.text}>
          {getString(
            suggestionBlock?.appliedCheckSum && suggestionBlock?.appliedCheckSum === checksum
              ? 'pr.suggestionApplied'
              : 'pr.suggestedChange'
          )}
        </Text>

        <Container>
          <Container className={css.removed}>
            <pre className={language}>
              <code className={`${language} code-highlight`} dangerouslySetInnerHTML={{ __html: html1 }}></code>
            </pre>
          </Container>
          <Container className={css.added}>
            <pre className={language}>
              <code className={`${language} code-highlight`} dangerouslySetInnerHTML={{ __html: html2 }}></code>
            </pre>
          </Container>
        </Container>
        {!!suggestionCheckSums?.length && (
          <Container data-section-id="CodeSuggestionBlockButtons">
            <Layout.Horizontal spacing="small" padding="medium">
              <Match expr={states.addedToBatch}>
                <Truthy>
                  <Button
                    intent="danger"
                    variation={ButtonVariation.SECONDARY}
                    text={getString('pr.removeSuggestion')}
                    onClick={actions.removeFromBatch}
                  />
                </Truthy>
                <Else>
                  <Button
                    variation={ButtonVariation.TERTIARY}
                    text={getString('pr.addSuggestion')}
                    onClick={actions.addToBatch}
                    disabled={!!states.otherAddedToBatch}
                  />
                  <Button
                    variation={ButtonVariation.TERTIARY}
                    text={getString('pr.commitSuggestion')}
                    onClick={actions.commit}
                    disabled={!!states.otherAddedToBatch}
                  />
                </Else>
              </Match>
            </Layout.Horizontal>
          </Container>
        )}
      </Layout.Vertical>
    </Container>
  )
}
