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

import React, { Fragment } from 'react'
import {
  Layout,
  Text,
  PageHeader,
  Utils,
  Avatar,
  FlexExpander,
  Container,
  Button,
  ButtonVariation,
  useToaster
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Link, useParams, useHistory } from 'react-router-dom'
import { Calendar, GitFork, Timer } from 'iconoir-react'
import { useMutate } from 'restful-react'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { CODEProps } from 'RouteDefinitions'
import type { GitInfoProps } from 'utils/GitUtils'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { getStatus } from 'utils/ExecutionUtils'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { getErrorMessage, timeDistance } from 'utils/Utils'
import useLiveTimer from 'hooks/useLiveTimeHook'
import { CommitActions } from 'components/CommitActions/CommitActions'
import type { TypesExecution } from 'services/code'
import { RepoArchivedBanner } from 'components/RepositoryArchivedBanner/RepositoryArchivedBanner'
import css from './ExecutionPageHeader.module.scss'

interface BreadcrumbLink {
  label: string
  url: string
}

interface ExecutionInfo {
  message: string
  authorName: string
  authorEmail: string
  source: string
  hash: string
  status: string
  started: number
  finished: number
}

interface ExecutionPageHeaderProps extends Optional<Pick<GitInfoProps, 'repoMetadata'>, 'repoMetadata'> {
  title: string | JSX.Element
  dataTooltipId: string
  extraBreadcrumbLinks?: BreadcrumbLink[]
  executionInfo?: ExecutionInfo
  pipeline: string
  execution: string
}

export function ExecutionPageHeader({
  repoMetadata,
  title,
  extraBreadcrumbLinks = [],
  executionInfo,
  pipeline,
  execution
}: ExecutionPageHeaderProps) {
  const { gitRef } = useParams<CODEProps>()
  const history = useHistory()
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { routes } = useAppContext()
  const currentTime = useLiveTimer()
  const { showSuccess, showError, clear: clearToaster } = useToaster()

  const { mutate: cancelExecution } = useMutate<TypesExecution>({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata?.path}/+/pipelines/${pipeline}/executions/${execution}/cancel`
  })

  const isActive = executionInfo?.status === ExecutionState.RUNNING

  if (!repoMetadata) {
    return null
  }

  return (
    <>
      <PageHeader
        className={css.pageHeader}
        title={title}
        breadcrumbs={
          <Layout.Horizontal
            spacing="small"
            className={css.breadcrumb}
            padding={{ bottom: 0 }}
            margin={{ bottom: 'small' }}>
            <Link to={routes.toCODERepositories({ space })}>{getString('repositories')}</Link>
            <Icon name="main-chevron-right" size={8} color={Color.GREY_500} />
            <Link to={routes.toCODERepository({ repoPath: repoMetadata.path as string, gitRef })}>
              {repoMetadata.identifier}
            </Link>
            {extraBreadcrumbLinks.map(link => (
              <Fragment key={link.url}>
                <Icon name="main-chevron-right" size={8} color={Color.GREY_500} />
                {/* This allows for outer most entities to not necessarily be links */}
                {link.url ? (
                  <Link to={link.url}>{link.label}</Link>
                ) : (
                  <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
                    {link.label}
                  </Text>
                )}
              </Fragment>
            ))}
          </Layout.Horizontal>
        }
        content={
          executionInfo && (
            <Container className={css.executionInfo}>
              <ExecutionStatus status={getStatus(executionInfo.status)} iconOnly noBackground iconSize={18} isCi />
              <Text inline lineClamp={1} color={Color.GREY_800} font={{ size: 'small' }}>
                {executionInfo.message}
              </Text>
              <PipeSeparator height={7} />
              <Avatar
                email={executionInfo.authorEmail}
                name={executionInfo.authorName}
                size="small"
                hoverCard={false}
              />
              <Text inline color={Color.GREY_500} font={{ size: 'small' }}>
                {executionInfo.authorName}
              </Text>
              <PipeSeparator height={7} />
              <GitFork height={12} width={12} color={Utils.getRealCSSColor(Color.GREY_500)} />
              <Text inline color={Color.GREY_500} font={{ size: 'small' }}>
                {executionInfo.source}
              </Text>
              <PipeSeparator height={7} />
              {executionInfo.hash && (
                <Container onClick={Utils.stopEvent}>
                  <CommitActions
                    href={routes.toCODECommit({
                      repoPath: repoMetadata.path as string,
                      commitRef: executionInfo.hash
                    })}
                    sha={executionInfo.hash}
                    enableCopy
                  />
                </Container>
              )}
              <FlexExpander />
              {executionInfo.started && (
                <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center' }} className={css.timer}>
                  <Timer height={16} width={16} color={Utils.getRealCSSColor(Color.GREY_500)} />
                  <Text inline color={Color.GREY_500} font={{ size: 'small' }}>
                    {isActive
                      ? timeDistance(executionInfo.started, currentTime, true) // Live update time when status is 'RUNNING'
                      : timeDistance(executionInfo.started, executionInfo.finished, true)}
                  </Text>
                  {executionInfo.finished && (
                    <>
                      <PipeSeparator height={7} />
                      <Calendar height={16} width={16} color={Utils.getRealCSSColor(Color.GREY_500)} />
                      <Text inline color={Color.GREY_500} font={{ size: 'small' }}>
                        {timeDistance(executionInfo.finished, currentTime, true)} ago
                      </Text>
                    </>
                  )}
                </Layout.Horizontal>
              )}
              <>
                <PipeSeparator height={7} />
                <Button
                  variation={ButtonVariation.PRIMARY}
                  text={getString('pipelines.edit')}
                  onClick={e => {
                    e.stopPropagation()
                    if (repoMetadata?.path && pipeline) {
                      history.push(routes.toCODEPipelineEdit({ repoPath: repoMetadata.path, pipeline }))
                    }
                  }}
                />
              </>
              {[ExecutionState.RUNNING, ExecutionState.PENDING].includes(getStatus(executionInfo?.status)) && (
                <>
                  <PipeSeparator height={7} />
                  <Button
                    variation={ButtonVariation.SECONDARY}
                    text={getString('cancel')}
                    onClick={async () => {
                      try {
                        await cancelExecution(null)
                        clearToaster()
                        showSuccess(getString('pipelines.executionCancelled'))
                      } catch (exception) {
                        showError(getErrorMessage(exception), 0, 'pipelines.executionCouldNotCancel')
                      }
                    }}
                  />
                </>
              )}
            </Container>
          )
        }
      />
      <RepoArchivedBanner isArchived={repoMetadata?.archived} updated={repoMetadata?.updated} />
    </>
  )
}
