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

import { useCallback, useEffect, useMemo, useState } from 'react'
import { useGet } from 'restful-react'
import { isEqual } from 'lodash-es'
import { useAtom, atom, useAtomValue } from 'jotai'
import { selectAtom } from 'jotai/utils'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import type { TypesPullReq, TypesPullReqActivity } from 'services/code'
import { usePRChecksDecision } from 'hooks/usePRChecksDecision'
import useSpaceSSE, { SSEEvents } from 'hooks/useSpaceSSE'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { PullRequestSection } from 'utils/Utils'

/**
 * This hook abstracts data handling for a pull request. It's used as a
 * centralized data store for all tabs in Pull Request page. The hook
 * fetches neccessary repository metadata, poll/refetch request metadata
 * for updates, cache data, etc...
 *
 * The abstraction allows Pull Request tabs to do less data handling and
 * focus more on their specific rendering logics.
 *
 * We use Atom to reduce React rendering cycles.
 */
export function useGetPullRequestInfo() {
  const space = useGetSpaceParam()
  const {
    repoMetadata,
    error: repoError,
    loading: repoLoading,
    refetch: refetchRepo,
    pullRequestId,
    pullRequestSection = PullRequestSection.CONVERSATION,
    commitSHA
  } = useGetRepositoryMetadata()
  const withActivities =
    pullRequestSection == PullRequestSection.CONVERSATION || pullRequestSection == PullRequestSection.FILES_CHANGED
  const {
    data: pullReqData,
    error: pullReqError,
    loading: pullReqLoading,
    refetch: refetchPullReq
  } = useGet<TypesPullReq>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pullreq/${pullRequestId}`,
    lazy: !repoMetadata
  })
  const [showEditDescription, setShowEditDescription] = useState(false)

  // Listen to PULLREQ_UPDATED event and refetch PR data accordingly
  useSpaceSSE({
    space,
    events: [SSEEvents.PULLREQ_UPDATED],
    onEvent: useCallback(
      (data: TypesPullReq) => {
        // Ensure this update belongs to the current PR
        if (data && String(data?.number) === pullRequestId) {
          // NOTE: We can't replace `pullReqMetadata` by `data` as events don't contain all
          // pr stats yet (can be optimized).
          refetchPullReq()
        }
      },
      [pullRequestId, refetchPullReq]
    )
  })

  const [pullReqMetadata, setPullReqMetadata] = useAtom(pullReqAtom)
  const pullReqStats = useAtomValue(pullReqStatsAtom)

  // TODO: Polling from usePRChecksDecision() starts React re-rendering check
  // Need a better way to handle (SSE, or atom in a smaller component that
  // writes latest decisions in a way that does not trigger re-rendering on
  // page level)
  const pullReqChecksDecision = usePRChecksDecision({
    repoMetadata,
    pullReqMetadata
  })

  const {
    data: activities,
    loading: activitiesLoading,
    error: activitiesError,
    refetch: refetchActivities
  } = useGet<TypesPullReqActivity[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pullreq/${pullRequestId}/activities`,
    lazy: true
  })
  const [pullReqActivities, setPullReqActivities] = useAtom(pullReqActivitiesAtom)

  const loading = useMemo(
    () => repoLoading || (pullReqLoading && !pullReqMetadata) || (activitiesLoading && !pullReqActivities),
    [repoLoading, pullReqLoading, pullReqMetadata, activitiesLoading, pullReqActivities]
  )

  useEffect(() => {
    if (activities) {
      setPullReqActivities(oldActivities => (isEqual(oldActivities, activities) ? oldActivities : activities))
    }
  }, [activities, setPullReqActivities])

  // Note: Activities are pulled automatically when pullReq's stats changed. Meaning when
  // there's an SSE event triggered, and stats come back different, then activities will
  // be re-fetched.
  useEffect(() => {
    if (withActivities && pullReqStats) {
      refetchActivities()
    }
  }, [withActivities, pullReqStats, refetchActivities])

  // Reset pullReqAtom to undefined when page is unmounted to make sure no
  // wrong caching occurs when navigating among PRs. This is important to make sure when
  // switching among PRs, cached data from atoms from one PR is not used for another
  useEffect(
    function cleanupAtoms() {
      return () => {
        setPullReqMetadata(undefined)
        setPullReqActivities(undefined)
      }
    },
    [setPullReqMetadata, setPullReqActivities]
  )

  // (1) pullReqMetadata holds the latest good PR data to make sure page is not broken
  // when polling fails.
  // (2) Only update pullReqMetadata when polled data is different from current one
  useEffect(
    function updatePullReqMetadata() {
      if (pullReqData && !isEqual(pullReqMetadata, pullReqData)) {
        setPullReqMetadata(pullReqData)
      }
    },
    [pullReqData, pullReqMetadata, setPullReqMetadata]
  )

  const retryOnErrorFunc = useMemo(() => {
    return () => (repoError ? refetchRepo() : pullReqError ? refetchPullReq() : refetchActivities())
  }, [repoError, pullReqError, activitiesError])

  return {
    repoMetadata,
    refetchRepo,
    loading,
    error: repoError || pullReqError || activitiesError,
    pullReqChecksDecision,
    showEditDescription,
    setShowEditDescription,
    pullReqMetadata,
    pullReqStats,
    pullRequestId,
    pullRequestSection,
    commitSHA,
    refetchActivities,
    retryOnErrorFunc
  }
}

export type UseGetPullRequestInfoResult = ReturnType<typeof useGetPullRequestInfo>

export function usePullReqActivities() {
  const [activities] = useAtom(pullReqActivitiesAtom)
  return activities
}

const pullReqAtom = atom<TypesPullReq | undefined>(undefined)
const pullReqStatsAtom = selectAtom(pullReqAtom, pullReq => pullReq?.stats, isEqual)
const pullReqActivitiesAtom = atom<TypesPullReqActivity[] | undefined>(undefined)
