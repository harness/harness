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
import type { TypesRepository, TypesPullReq, TypesCheck } from 'services/code'
import { useStrings } from 'framework/strings'
import { CheckStatus, PullRequestSection, timeDistance } from 'utils/Utils'
import { usePRChecksDecision } from 'hooks/usePRChecksDecision3'
import css from '../PullRequestOverviewPanel.module.scss'

interface ChecksSectionProps {
  repoMetadata: TypesRepository
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
    const statusCounts = { failed: 0, pending: 0, running: 0, succeeded: 0, total: checks?.length || 0 }
    if (isEmpty(checks)) {
      return { message: '', summary: statusCounts }
    }

    // Count occurrences of each status
    checks.forEach(check => {
      const status = check.check.status
      if (status === CheckStatus.FAILURE || status === CheckStatus.ERROR) {
        statusCounts.failed += 1
      } else if (status === CheckStatus.PENDING) {
        statusCounts.pending += 1
      } else if (status === CheckStatus.RUNNING) {
        statusCounts.running += 1
      } else if (status === CheckStatus.SUCCESS) {
        statusCounts.succeeded += 1
      }
    })

    // Format the summary string
    const summaryParts = []
    if (statusCounts.failed > 0) {
      summaryParts.push(`${statusCounts.failed} failed`)
    }
    if (statusCounts.pending > 0) {
      summaryParts.push(`${statusCounts.pending} pending`)
    }
    if (statusCounts.running > 0) {
      summaryParts.push(`${statusCounts.running} running`)
    }
    if (statusCounts.succeeded > 0) {
      summaryParts.push(`${statusCounts.succeeded} succeeded`)
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
        color = Color.RED_700
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
      <Container className={cx(css.sectionContainer, css.borderContainer)}>
        <Container>
          <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
            <Layout.Horizontal flex={{ alignItems: 'center' }}>
              <StatusCircle summary={generateStatusSummary(data?.checks as TypeCheckData[])} />
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
              padding={{ bottom: 'medium' }}
              className={cx(css.showMore, css.blueText, css.buttonPadding)}
              variation={ButtonVariation.LINK}
              size={ButtonSize.SMALL}
              text={getString(isExpanded ? 'showLessCheck' : 'showCheckAll')}
              onClick={toggleExpanded}
            />
          </Layout.Horizontal>
        </Container>
      </Container>
      <Render when={isExpanded && data}>
        {data?.checks?.map(check => {
          return (
            <Container
              key={check?.check?.id}
              className={cx(css.borderContainer, css.greyContainer)}
              padding={{ left: 'xxlarge' }}>
              <Layout.Horizontal padding={{ left: 'xsmall' }} flex={{ justifyContent: 'space-between' }}>
                <Layout.Horizontal
                  padding={{ top: 'xsmall', bottom: 'xsmall' }}
                  flex={{ alignItems: 'center', justifyContent: 'space-around' }}>
                  <ExecutionStatus
                    status={check.check.status as ExecutionState}
                    iconOnly
                    noBackground
                    iconSize={16}
                    isCi
                  />
                  <Text
                    padding={{ left: 'small', top: 'xsmall', bottom: 'xsmall' }}
                    width={200}
                    lineClamp={1}
                    color={Color.GREY_700}
                    className={cx(css.checkName, css.textSize)}
                    font={{ variation: FontVariation.BODY }}>
                    {check.check.uid}
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
                        time: <>{timeDistance(check.check.created, check.check.updated)}</>
                      }}
                    />
                  </Text>
                </Layout.Horizontal>
                <Container padding={{ right: 'xxxlarge' }}>
                  <Layout.Horizontal flex={{ justifyContent: 'center' }}>
                    {check.required && (
                      <Container className={css.requiredContainer}>
                        <Text className={css.requiredText}>{getString('required')}</Text>
                      </Container>
                    )}
                    <Link
                      to={
                        routes.toCODEPullRequest({
                          repoPath: repoMetadata.path as string,
                          pullRequestId: String(pullReqMetadata.number),
                          pullRequestSection: PullRequestSection.CHECKS
                        }) + `?uid=${check.check.uid}`
                      }>
                      <Text padding={{ left: 'medium' }} color={Color.PRIMARY_7} className={css.blueText}>
                        {getString('details')}
                      </Text>
                    </Link>
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
// Define the props type
interface CircleSegmentProps {
  radius: number
  stroke: number
  color: string
  offset: string | number
  strokeDasharray: string
}

// Type the CircleSegment functional component with CircleSegmentProps
const CircleSegment: React.FC<CircleSegmentProps> = ({ radius, stroke, color, offset, strokeDasharray }) => {
  return (
    <circle
      r={radius}
      cx="12.5" // Center of the SVG in the 30x30 viewBox
      cy="12.5" // Center of the SVG in the 30x30 viewBox
      fill="transparent"
      stroke={color}
      strokeWidth={stroke}
      strokeDasharray={strokeDasharray}
      strokeDashoffset={offset}
    />
  )
}

const StatusCircle = ({
  summary
}: {
  summary: {
    message: string
    summary: {
      failed: number
      pending: number
      running: number
      succeeded: number
      total: number
    }
  }
}) => {
  const data = summary.summary
  const radius = 10 // Adjusted radius to fit in 30x30 icon, assuming you want it larger
  const circumference = 2 * Math.PI * radius
  const stroke = 5
  // Calculate the dasharray percentages for each status
  const failedPercentage = (data.failed / data.total) * circumference
  const pendingPercentage = (data.pending / data.total) * circumference
  const runningPercentage = (data.running / data.total) * circumference
  const succeededPercentage = (data.succeeded / data.total) * circumference

  return (
    <svg width="25" height="25" viewBox="0 0 25 25">
      <g transform={`rotate(-90 12.5 12.5)`}>
        <CircleSegment
          radius={radius}
          stroke={stroke}
          color="#cf2318"
          strokeDasharray={`${failedPercentage} ${circumference}`}
          offset="0"
        />
        <CircleSegment
          radius={radius}
          stroke={stroke}
          color="orange"
          strokeDasharray={`${pendingPercentage} ${circumference}`}
          offset={-failedPercentage}
        />
        <CircleSegment
          radius={radius}
          stroke={stroke}
          color="#0092e4"
          strokeDasharray={`${runningPercentage} ${circumference}`}
          offset={-(failedPercentage + pendingPercentage)}
        />
        <CircleSegment
          radius={radius}
          stroke={stroke}
          color="green"
          strokeDasharray={`${succeededPercentage} ${circumference}`}
          offset={-(failedPercentage + pendingPercentage + runningPercentage)}
        />
      </g>
    </svg>
  )
}
