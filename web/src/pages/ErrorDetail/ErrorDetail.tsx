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

import React, { useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Button, ButtonVariation, Container, Layout, PageBody, Text, TextInput, useToaster } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useGet, useMutate } from 'restful-react'
import { Render } from 'react-jsx-match'
import { isEmpty } from 'lodash-es'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { StringKeys, useStrings } from 'framework/strings'
import type { CODEProps } from 'RouteDefinitions'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { StatusBadge } from 'components/StatusBadge/StatusBadge'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { getErrorMessage, voidFn } from 'utils/Utils'
import { useTaskNumbersByOriginRef } from 'utils/TaskUtils'
import {
  INSIGHTS_STATUS_COLORS,
  InsightsException,
  InsightsIssueStatus,
  InsightsIssueUpdateInput,
  TypesInsightsEvent,
  TypesInsightsIssue,
  getInsightsApiPath,
  getLevelColors,
  parseEventExceptions
} from 'utils/InsightsUtils'
import css from './ErrorDetail.module.scss'

const EVENTS_LIMIT = 10

export default function ErrorDetail() {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const { showError, showSuccess } = useToaster()
  const { issueId } = useParams<CODEProps>()
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const repoPath = repoMetadata?.path as string
  const issuePath = `${getInsightsApiPath(repoMetadata?.path)}/issues/${issueId}`
  const [release, setRelease] = useState('')
  const taskNumbers = useTaskNumbersByOriginRef(repoMetadata?.path, 'insights')

  const {
    data: issue,
    error: issueError,
    loading: issueLoading,
    refetch: refetchIssue
  } = useGet<TypesInsightsIssue>({ path: issuePath, lazy: !repoMetadata })
  const { data: events, loading: eventsLoading } = useGet<TypesInsightsEvent[]>({
    path: `${issuePath}/events`,
    queryParams: { page: 1, limit: EVENTS_LIMIT },
    lazy: !repoMetadata
  })
  const { mutate: updateIssue, loading: updating } = useMutate<TypesInsightsIssue>({ verb: 'PATCH', path: issuePath })

  const setStatus = (status: InsightsIssueStatus) => {
    const input: InsightsIssueUpdateInput = { status }
    if (status === 'resolved' && release.trim()) {
      input.resolved_in_release = release.trim()
    }
    updateIssue(input)
      .then(() => {
        showSuccess(getString('insights.issueUpdated'))
        refetchIssue()
      })
      .catch(err => showError(getErrorMessage(err)))
  }

  const taskNumber = issue ? taskNumbers[`${issue.project_id}:${issue.fingerprint}`] : undefined

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={issue?.title || getString('pageTitle.errorDetail')}
        dataTooltipId="repositoryErrorDetail"
        extraBreadcrumbLinks={
          repoMetadata
            ? [
                { label: getString('insights.title'), url: routes.toCODEErrors({ repoPath }) },
                { label: `#${issueId}`, url: '' }
              ]
            : undefined
        }
      />
      <PageBody error={getErrorMessage(error || issueError)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading || issueLoading || eventsLoading} withBorder />

        <Render when={issue}>
          <Layout.Vertical spacing="medium" padding="xlarge">
            <Layout.Horizontal spacing="medium" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
              <StatusBadge label={issue?.level} colors={getLevelColors(issue?.level)} />
              {issue?.status && (
                <StatusBadge
                  label={getString(`insights.statusValue.${issue.status}` as StringKeys)}
                  colors={INSIGHTS_STATUS_COLORS[issue.status]}
                />
              )}
              {taskNumber ? (
                <Link to={routes.toCODETask({ repoPath, taskNumber: String(taskNumber) })}>TASK-{taskNumber}</Link>
              ) : null}
            </Layout.Horizontal>

            <Text font={{ variation: FontVariation.H4 }}>{issue?.title}</Text>
            <Text color={Color.GREY_600}>{issue?.culprit}</Text>

            <div className={css.stats}>
              <Stat label={getString('insights.timesSeen')} value={String(issue?.times_seen ?? 0)} />
              <Stat label={getString('insights.usersSeen')} value={String(issue?.users_seen ?? 0)} />
              <Stat label={getString('insights.firstRelease')} value={issue?.first_release || '-'} />
              <Stat label={getString('insights.lastRelease')} value={issue?.last_release || '-'} />
              <Stat label={getString('insights.resolvedInRelease')} value={issue?.resolved_in_release || '-'} />
              <div>
                <Text color={Color.GREY_500}>{getString('insights.lastSeen')}</Text>
                {issue?.last_seen ? <TimePopoverWithLocal time={issue.last_seen} inline={false} /> : <Text>-</Text>}
              </div>
            </div>

            <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
              <TextInput
                placeholder={getString('insights.resolvedInReleasePlaceholder')}
                value={release}
                onChange={e => setRelease((e.target as HTMLInputElement).value)}
              />
              <Button
                variation={ButtonVariation.PRIMARY}
                text={getString('insights.resolve')}
                disabled={updating || issue?.status === 'resolved'}
                onClick={() => setStatus('resolved')}
              />
              <Button
                variation={ButtonVariation.SECONDARY}
                text={getString('insights.ignore')}
                disabled={updating || issue?.status === 'ignored'}
                onClick={() => setStatus('ignored')}
              />
              <Render when={issue?.status === 'resolved' || issue?.status === 'ignored'}>
                <Button
                  variation={ButtonVariation.TERTIARY}
                  text={getString('insights.reopen')}
                  disabled={updating}
                  onClick={() => setStatus('unresolved')}
                />
              </Render>
            </Layout.Horizontal>

            <Text font={{ variation: FontVariation.H5 }}>{getString('insights.latestEvents')}</Text>
            {isEmpty(events) ? (
              <Text>{getString('insights.noEvents')}</Text>
            ) : (
              events?.map(event => <EventCard key={event.id} event={event} />)
            )}
          </Layout.Vertical>
        </Render>
      </PageBody>
    </Container>
  )
}

const Stat: React.FC<{ label: string; value: string }> = ({ label, value }) => (
  <div>
    <Text color={Color.GREY_500}>{label}</Text>
    <Text>{value}</Text>
  </div>
)

const EventCard: React.FC<{ event: TypesInsightsEvent }> = ({ event }) => {
  const { getString } = useStrings()
  const exceptions = useMemo(() => parseEventExceptions(event.payload), [event.payload])

  return (
    <Container className={css.event}>
      <Layout.Horizontal spacing="medium" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
        {event.timestamp ? <TimePopoverWithLocal time={event.timestamp} inline={false} /> : null}
        <StatusBadge label={event.level} colors={getLevelColors(event.level)} />
        {event.release ? <Text>{getString('insights.releaseValue', { release: event.release })}</Text> : null}
        {event.environment ? <Text color={Color.GREY_500}>{event.environment}</Text> : null}
      </Layout.Horizontal>
      {event.message ? <Text className={css.message}>{event.message}</Text> : null}
      {exceptions.map((exception, index) => (
        <ExceptionView key={index} exception={exception} />
      ))}
    </Container>
  )
}

const ExceptionView: React.FC<{ exception: InsightsException }> = ({ exception }) => (
  <div className={css.exception}>
    <Text font={{ weight: 'semi-bold' }} color={Color.RED_700}>
      {exception.type}
      {exception.value ? `: ${exception.value}` : ''}
    </Text>
    <ul className={css.frames}>
      {exception.frames.map((frame, index) => (
        <li key={index} className={frame.inApp ? css.frameInApp : css.frame}>
          <span className={css.frameLocation}>
            {frame.filename}
            {frame.lineno !== undefined ? `:${frame.lineno}` : ''}
          </span>
          {frame.function ? <span className={css.frameFunction}> {frame.function}</span> : null}
        </li>
      ))}
    </ul>
  </div>
)
