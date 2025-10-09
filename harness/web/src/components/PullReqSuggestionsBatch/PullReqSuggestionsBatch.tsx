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

import React, { useEffect, useMemo, useRef } from 'react'
import { Button, ButtonVariation, Container, stringSubstitute } from '@harnessio/uicore'
import { Render } from 'react-jsx-match'
import { useAtom } from 'jotai'
import { Suggestion, pullReqSuggestionsAtom } from 'atoms/pullReqSuggestions'
import { useStrings } from 'framework/strings'
import {
  useCommitPullReqSuggestions,
  useCommitSuggestionsModal
} from 'components/CommitModalButton/useCommitSuggestionModal'
import { PullRequestSection, getErrorMessage, waitUntil } from 'utils/Utils'
import { dispatchCustomEvent } from 'hooks/useEventListener'
import { PullReqCustomEvent, getActivePullReqPageSection } from 'pages/PullRequest/PullRequestUtils'
import css from './PullReqSuggestionsBatch.module.scss'

export const PullReqSuggestionsBatch: React.FC = () => {
  const [suggestions, setSuggestions] = useAtom(pullReqSuggestionsAtom)
  const suggestionsRef = useRef<Suggestion[]>(suggestions)
  const { getString } = useStrings()
  const text = useMemo(
    () => stringSubstitute(getString('pr.commitSuggestions'), { count: suggestions?.length }),
    [suggestions, getString]
  )
  const commitPullReqSuggestions = useCommitPullReqSuggestions()
  const [openCommitSuggestionsModal] = useCommitSuggestionsModal({
    title: text as string,
    commitMessage: stringSubstitute(getString('pr.applySuggestions'), { count: suggestions?.length }) as string,
    onCommit: async formData => {
      return new Promise(resolve => {
        commitPullReqSuggestions({
          bypass_rules: true,
          dry_run_rules: false,
          title: formData.commitMessage,
          message: formData.extendedDescription,
          suggestions: suggestionsRef.current
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
    suggestionsRef.current = suggestions
  }, [suggestions])

  useEffect(() => {
    setSuggestions([])
  }, [setSuggestions])

  return (
    <Render when={suggestions?.length}>
      <Container flex={{ alignItems: 'center' }}>
        <Button variation={ButtonVariation.TERTIARY} text={text} onClick={openCommitSuggestionsModal}>
          <span className={css.count}>{suggestions?.length}</span>
        </Button>
      </Container>
    </Render>
  )
}
