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
import {
  Button,
  ButtonSize,
  ButtonVariation,
  Container,
  Layout,
  useToggle,
  Text,
  StringSubstitute
} from '@harnessio/uicore'
import React, { useEffect, useState } from 'react'
import cx from 'classnames'
import { Render } from 'react-jsx-match'
import { Link } from 'react-router-dom'
import { Color, FontVariation } from '@harnessio/design-system'
import { isEmpty } from 'lodash-es'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { useAppContext } from 'AppContext'
import type { RepoRepositoryOutput, TypesPullReq, TypesCheck } from 'services/code'
import { useStrings } from 'framework/strings'
import { CheckStatus, PullRequestSection, timeDistance } from 'utils/Utils'
import { usePRChecksDecision } from 'hooks/usePRChecksDecision3'
import StatusCircle from './StatusCircle'
import Timeout from '../../../../../icons/code-timeout.svg?url'
import css from '../PullRequestOverviewPanel.module.scss'

interface ChecksSectionProps {
  repoMetadata: RepoRepositoryOutput
  pullReqMetadata: TypesPullReq
}

interface TypeCheckData {
  bypassable: boolean
  required: boolean
  check: TypesCheck
}
const ChecksSection = (props: ChecksSectionProps) => {
  const { repoMetadata, pullReqMetadata } = props
  const [isExpanded, toggleExpanded] = useToggle(false)
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const [statusMessage, setStatusMessage] = useState<{ color: string; title: string; content: string }>({
    color: '',
    title: '',
    content: ''
  })
  const { data, error } = usePRChecksDecision({ pullReqMetadata, repoMetadata })
  useShowRequestError(error)
  const [prData, setPrData] = useState<TypeCheckData[]>()

  function generateStatusSummary(checks: TypeCheckData[]) {
    // Initialize counts for each status
    const statusCounts = {
      failedReq: 0,
      pendingReq: 0,
      runningReq: 0,
      successReq: 0,
      failed: 0,
      pending: 0,
      running: 0,
      succeeded: 0,
      total: checks?.length || 0
    }
    if (isEmpty(checks)) {
      return { message: '', summary: statusCounts }
    }

    // Count occurrences of each status
    checks.forEach(check => {
      const status = check.check.status
      const required = check.required
      if (status === CheckStatus.FAILURE || status === CheckStatus.ERROR) {
        if (required) {
          statusCounts.failedReq += 1
        } else {
          statusCounts.failed += 1
        }
      } else if (status === CheckStatus.PENDING) {
        if (required) {
          statusCounts.pendingReq += 1
        } else {
          statusCounts.pending += 1
        }
      } else if (status === CheckStatus.RUNNING) {
        if (required) {
          statusCounts.runningReq += 1
        } else {
          statusCounts.running += 1
        }
      } else if (status === CheckStatus.SUCCESS) {
        if (required) {
          statusCounts.successReq += 1
        } else {
          statusCounts.succeeded += 1
        }
      }
    })

    // Format the summary string
    const summaryParts = []
    if (statusCounts.failed > 0 || statusCounts.failedReq) {
      const num = statusCounts.failed + statusCounts.failedReq
      summaryParts.push(`${num} failed`)
    }
    if (statusCounts.pending > 0 || statusCounts.pendingReq > 0) {
      const num = statusCounts.pending + statusCounts.pendingReq
      summaryParts.push(`${num} pending`)
    }
    if (statusCounts.running > 0 || statusCounts.runningReq) {
      const num = statusCounts.running + statusCounts.runningReq
      summaryParts.push(`${num} running`)
    }
    if (statusCounts.succeeded > 0 || statusCounts.successReq) {
      const num = statusCounts.succeeded + statusCounts.successReq
      summaryParts.push(`${num} succeeded`)
    }

    return { message: summaryParts.join(', '), summary: statusCounts }
  }
  function checkRequired(checkData: TypeCheckData[]) {
    return checkData.some(item => item.required)
  }

  function determineStatusMessage(
    checks: TypeCheckData[]
  ): { title: string; content: string; color: string } | undefined {
    if (checks === undefined || isEmpty(checks)) {
      return undefined
    }
    const { message } = generateStatusSummary(checks)
    let title = ''

    let content = ''
    let color = ''
    const oneRuleHasRequired = checkRequired(checks)
    if (oneRuleHasRequired) {
      if (
        checks.some(
          check =>
            (check.required && check.check.status === CheckStatus.FAILURE) ||
            (check.required && check.check.status === CheckStatus.ERROR)
        )
      ) {
        title = getString('checkSection.someReqChecksFailed')
        content = `${message}`
        color = Color.RED_700
      } else if (checks.some(check => check.required && check.check.status === CheckStatus.PENDING)) {
        title = getString('checkSection.someReqChecksPending')

        content = `${message}`
        color = Color.ORANGE_500
      } else if (checks.some(check => check.required && check.check.status === CheckStatus.RUNNING)) {
        title = getString('checkSection.someReqChecksRunning')

        content = `${message}`
        color = Color.PRIMARY_6
      } else if (checks.every(check => check.check.status === CheckStatus.SUCCESS)) {
        title = getString('checkSection.allChecksSucceeded')
        content = `${message}`
        color = Color.GREY_600
      } else {
        title = getString('checkSection.allReqChecksPassed')
        content = `${message}`
        color = Color.GREEN_800
      }
    } else {
      if (
        checks.some(check => check.check.status === CheckStatus.FAILURE || check.check.status === CheckStatus.ERROR)
      ) {
        title = getString('checkSection.someChecksFailed')
        content = ` ${message}`
        color = Color.GREY_600
      } else if (checks.some(check => check.check.status === CheckStatus.PENDING)) {
        title = getString('checkSection.someChecksNotComplete')

        content = `${message}`
        color = Color.ORANGE_500
      } else if (checks.some(check => check.check.status === CheckStatus.RUNNING)) {
        title = getString('checkSection.someChecksRunning')
        content = `${message}`
        color = Color.PRIMARY_6
      } else if (checks.every(check => check.check.status === CheckStatus.SUCCESS)) {
        title = getString('checkSection.allChecksSucceeded')
        content = `${message}`
        color = Color.GREY_600
      }
    }
    return { title, content, color }
  }

  useEffect(() => {
    if (data && !isEmpty(data.checks)) {
      setPrData(data.checks)
    }
  }, [data, repoMetadata, pullReqMetadata])

  useEffect(() => {
    if (prData) {
      const curStatusMessage = determineStatusMessage(prData)
      setStatusMessage(curStatusMessage || { color: '', title: '', content: '' })
    }
  }, [prData]) // eslint-disable-line react-hooks/exhaustive-deps

  return !isEmpty(data?.checks) ? (
    <Render when={!isEmpty(data?.checks)}>
      <Container
        className={cx(css.sectionContainer, css.borderContainer, { [css.mergedContainer]: pullReqMetadata.merged })}>
        <Container>
          <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
            <Layout.Horizontal flex={{ alignItems: 'center' }}>
              <Container>
                <StatusCircle summary={generateStatusSummary(data?.checks as TypeCheckData[])} />
              </Container>
              <Layout.Vertical padding={{ left: 'medium' }}>
                <Text
                  padding={{ bottom: 'xsmall' }}
                  className={css.sectionTitle}
                  color={(statusMessage as { color: string }).color}>
                  {statusMessage.title}
                </Text>
                <Text className={css.sectionSubheader} color={Color.GREY_450} font={{ variation: FontVariation.BODY }}>
                  {statusMessage.content}
                </Text>
              </Layout.Vertical>
            </Layout.Horizontal>
            <Button
              className={cx(css.blueText, css.buttonPadding)}
              variation={ButtonVariation.LINK}
              size={ButtonSize.SMALL}
              text={getString(isExpanded ? 'showLessMatches' : 'showMoreText')}
              onClick={toggleExpanded}
              rightIcon={isExpanded ? 'main-chevron-up' : 'main-chevron-down'}
              iconProps={{ size: 10, margin: { left: 'xsmall' } }}
            />
          </Layout.Horizontal>
        </Container>
      </Container>
      <Render when={isExpanded && data}>
        {data?.checks?.map(check => {
          return (
            // eslint-disable-next-line react/jsx-key
            <Container className={cx(css.borderContainer, css.greyContainer)} padding={{ left: 'xxlarge' }}>
              <Layout.Horizontal padding={{ left: 'xxlarge' }} flex={{ justifyContent: 'space-between' }}>
                <Layout.Horizontal
                  padding={{ top: 'xsmall', bottom: 'xsmall', left: 'xsmall' }}
                  flex={{ alignItems: 'center', justifyContent: 'space-around' }}>
                  {check.check.status === ExecutionState.PENDING ? (
                    <Container className={css.pendingIcon}>
                      <img
                        alt={getString('waiting')}
                        width={18}
                        height={18}
                        src={Timeout}
                        className={css.timeoutIcon}
                      />
                    </Container>
                  ) : (
                    <ExecutionStatus
                      status={check.check.status as ExecutionState}
                      iconOnly
                      noBackground
                      iconSize={check.check.status === 'failure' ? 17 : 16}
                      isCi
                      inPr
                    />
                  )}
                  <Text
                    padding={{ left: 'small', top: 'xsmall', bottom: 'xsmall' }}
                    width={200}
                    lineClamp={1}
                    color={Color.GREY_700}
                    className={cx(css.checkName, css.textSize)}
                    font={{ variation: FontVariation.BODY }}>
                    {check.check.identifier}
                  </Text>
                  <Text
                    padding={{ left: 'small' }}
                    width={200}
                    lineClamp={1}
                    color={Color.GREY_400}
                    className={cx(css.checkName, css.textSize)}
                    font={{ variation: FontVariation.BODY }}>
                    <StringSubstitute
                      str={getString(
                        check.check.status === CheckStatus.SUCCESS
                          ? 'checkStatus.succeeded'
                          : check.check.status === CheckStatus.FAILURE
                          ? 'checkStatus.failed'
                          : check.check.status === CheckStatus.RUNNING
                          ? 'checkStatus.running'
                          : check.check.status === CheckStatus.PENDING
                          ? 'checkStatus.pending'
                          : 'checkStatus.error'
                      )}
                      vars={{
                        time: <>{timeDistance(check.check.started, check.check.ended)}</>
                      }}
                    />
                  </Text>
                </Layout.Horizontal>
                <Container className={check.required ? css.checkContainerPadding : css.paddingWithOutReq}>
                  <Layout.Horizontal className={css.gridContainer} flex={{ justifyContent: 'center' }}>
                    {check.check.status !== CheckStatus.PENDING && (
                      <Link
                        className={cx(css.details, css.gridItem)}
                        to={
                          routes.toCODEPullRequest({
                            repoPath: repoMetadata.path as string,
                            pullRequestId: String(pullReqMetadata.number),
                            pullRequestSection: PullRequestSection.CHECKS
                          }) + `?uid=${check.check.identifier}`
                        }>
                        <Text padding={{ left: 'medium' }} color={Color.PRIMARY_7} className={css.blueText}>
                          {getString('details')}
                        </Text>
                      </Link>
                    )}
                    {check.required && (
                      <Container className={cx(css.requiredContainer, css.required, css.gridItem)}>
                        <Text className={css.requiredText}>{getString('required')}</Text>
                      </Container>
                    )}
                  </Layout.Horizontal>
                </Container>
              </Layout.Horizontal>
            </Container>
          )
        })}
      </Render>
    </Render>
  ) : null
}

export default ChecksSection
