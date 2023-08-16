import React from 'react'
import { Falsy, Match, Truthy } from 'react-jsx-match'
import {
  Container,
  Layout,
  Text,
  useToggle,
  FontVariation,
  Button,
  ButtonVariation,
  ButtonSize,
  Utils
} from '@harness/uicore'
import { Link } from 'react-router-dom'
import type { GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { PRCheckExecutionStatus, PRCheckExecutionState } from 'components/PRCheckExecutionStatus/PRCheckExecutionStatus'
import { useShowRequestError } from 'hooks/useShowRequestError'
import type { TypesCheck } from 'services/code'
import { useAppContext } from 'AppContext'
import { PullRequestSection, timeDistance } from 'utils/Utils'
import type { PRChecksDecisionResult } from 'hooks/usePRChecksDecision'
import css from './ChecksOverview.module.scss'

interface ChecksOverviewProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  prChecksDecisionResult: PRChecksDecisionResult
}

export function ChecksOverview({ repoMetadata, pullRequestMetadata, prChecksDecisionResult }: ChecksOverviewProps) {
  const { getString } = useStrings()
  const [isExpanded, toggleExpanded] = useToggle(false)
  const { data, overallStatus, error, color, message } = prChecksDecisionResult

  useShowRequestError(error)

  return overallStatus && data?.length ? (
    <Container
      className={css.main}
      margin={{ bottom: pullRequestMetadata.description ? undefined : 'large' }}
      style={{ '--border-color': Utils.getRealCSSColor(color) } as React.CSSProperties}>
      <Match expr={isExpanded}>
        <Truthy>
          <Container className={css.checks}>
            <Layout.Vertical spacing="medium">
              {/* <CheckSection
                repoMetadata={repoMetadata}
                pullRequestMetadata={pullRequestMetadata}
                data={data}
                isPipeline
              /> */}
              <CheckSection
                repoMetadata={repoMetadata}
                pullRequestMetadata={pullRequestMetadata}
                data={data}
                isPipeline={false}
              />
            </Layout.Vertical>
          </Container>
        </Truthy>
        <Falsy>
          <Layout.Horizontal spacing="small" className={css.layout}>
            <PRCheckExecutionStatus status={overallStatus} noBackground iconOnly />
            <Text font={{ variation: FontVariation.LEAD }}>{getString('pr.checks')}</Text>
            <Text color={color} padding={{ left: 'small' }} font={{ variation: FontVariation.FORM_MESSAGE_WARNING }}>
              {message}
            </Text>
          </Layout.Horizontal>
        </Falsy>
      </Match>
      <Button
        className={css.showMore}
        variation={ButtonVariation.LINK}
        size={ButtonSize.SMALL}
        text={getString(isExpanded ? 'showLess' : 'showMore')}
        rightIcon={isExpanded ? 'main-chevron-up' : 'main-chevron-down'}
        iconProps={{ size: 10, margin: { left: 'xsmall' } }}
        onClick={toggleExpanded}
      />
    </Container>
  ) : null
}

interface CheckSectionProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  data: TypesCheck[]
  isPipeline: boolean
}

const CheckSection: React.FC<CheckSectionProps> = ({ repoMetadata, pullRequestMetadata, data, isPipeline }) => {
  const { getString } = useStrings()
  const { routes } = useAppContext()

  return data.length ? (
    <Container>
      <Layout.Vertical spacing="small">
        <Text
          icon={isPipeline ? 'pipeline' : 'main-search'}
          iconProps={{ size: 12, padding: { right: 'small' } }}
          font={{ variation: FontVariation.SMALL_BOLD }}>
          {getString(isPipeline ? 'pageTitle.pipelines' : 'checks')}
        </Text>
        <Container className={css.table}>
          {data.map(({ uid, status, summary, created, updated }) => (
            <Container className={css.row} key={uid}>
              <Layout.Horizontal className={css.rowLayout}>
                <Container className={css.status}>
                  <PRCheckExecutionStatus status={status as PRCheckExecutionState} />
                </Container>

                <Link
                  to={
                    routes.toCODEPullRequest({
                      repoPath: repoMetadata.path as string,
                      pullRequestId: String(pullRequestMetadata.number),
                      pullRequestSection: PullRequestSection.CHECKS
                    }) + `?uid=${uid}`
                  }>
                  <Text font={{ variation: FontVariation.SMALL_BOLD }} className={css.name} lineClamp={1}>
                    {uid}
                  </Text>
                </Link>

                <Text className={css.desc} font={{ variation: FontVariation.SMALL }} lineClamp={1}>
                  {summary}
                </Text>

                <Text className={css.time} font={{ variation: FontVariation.SMALL }}>
                  {timeDistance(updated, created)}
                </Text>
              </Layout.Horizontal>
            </Container>
          ))}
        </Container>
      </Layout.Vertical>
    </Container>
  ) : null
}
