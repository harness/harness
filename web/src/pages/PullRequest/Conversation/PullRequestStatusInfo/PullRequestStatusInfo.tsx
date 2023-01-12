import React from 'react'
import {
  Button,
  Color,
  Container,
  FlexExpander,
  Icon,
  Layout,
  StringSubstitute,
  Text,
  useToaster
} from '@harness/uicore'
import { useMutate } from 'restful-react'
import cx from 'classnames'
import ReactTimeago from 'react-timeago'
import type { TypesPullReq } from 'services/code'
import { useStrings } from 'framework/strings'
import { CodeIcon, GitInfoProps, PullRequestFilterOption } from 'utils/GitUtils'
import { getErrorMessage } from 'utils/Utils'
import css from './PullRequestStatusInfo.module.scss'

interface PullRequestStatusInfoProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  onMerge: () => void
}

export const PullRequestStatusInfo: React.FC<PullRequestStatusInfoProps> = ({
  repoMetadata,
  pullRequestMetadata,
  onMerge
}) => {
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { mutate: mergePR } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/merge`
  })

  if (pullRequestMetadata.state === PullRequestFilterOption.MERGED) {
    return <MergeInfo pullRequestMetadata={pullRequestMetadata} />
  }

  return (
    <Container className={css.main} padding="xlarge">
      <Layout.Vertical spacing="xlarge">
        <Container>
          <Layout.Horizontal spacing="medium" flex={{ alignItems: 'center' }}>
            <Icon name="tick-circle" size={28} color={Color.GREEN_700} />
            <Container>
              <Text className={css.heading}>{getString('pr.branchHasNoConflicts')}</Text>
              <Text className={css.sub}>{getString('pr.prCanBeMerged')}</Text>
            </Container>
            <FlexExpander />
          </Layout.Horizontal>
        </Container>
        <Container>
          <Button
            className={css.btn}
            text={getString('pr.mergePR')}
            onClick={() => {
              mergePR({})
                .then(onMerge)
                .catch(exception => showError(getErrorMessage(exception)))
            }}
          />
        </Container>
      </Layout.Vertical>
    </Container>
  )
}

const MergeInfo: React.FC<{ pullRequestMetadata: TypesPullReq }> = ({ pullRequestMetadata }) => {
  const { getString } = useStrings()

  return (
    <Container className={cx(css.main, css.merged)} padding="xlarge">
      <Layout.Horizontal spacing="medium" flex={{ alignItems: 'center' }}>
        <Icon name={CodeIcon.PullRequest} size={28} color={Color.PURPLE_700} />
        <Container>
          <Text className={css.heading}>{getString('pr.prMerged')}</Text>
          <Text className={css.sub}>
            <StringSubstitute
              str={getString('pr.prMergedInfo')}
              vars={{
                user: <strong>{pullRequestMetadata.merger?.display_name}</strong>,
                source: <strong>{pullRequestMetadata.source_branch}</strong>,
                target: <strong>{pullRequestMetadata.target_branch} </strong>,
                time: <ReactTimeago date={pullRequestMetadata.merged as number} />
              }}
            />
          </Text>
        </Container>
        <FlexExpander />
      </Layout.Horizontal>
    </Container>
  )
}
