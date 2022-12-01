import React from 'react'
import { Button, Container, ButtonVariation, PageBody } from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { CodeIcon, makeDiffRefs } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { getErrorMessage } from 'utils/Utils'
import emptyStateImage from 'images/empty-state.svg'
import css from './PullRequests.module.scss'

export default function PullRequests() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('pullRequests')}
        dataTooltipId="repositoryPullRequests"
      />
      <PageBody
        loading={loading}
        error={getErrorMessage(error)}
        retryOnError={() => refetch()}
        noData={{
          when: () => repoMetadata !== null,
          message: getString('pullRequestEmpty'),
          image: emptyStateImage,
          button: (
            <Button
              variation={ButtonVariation.PRIMARY}
              text={getString('newPullRequest')}
              icon={CodeIcon.Add}
              onClick={() => {
                history.push(
                  routes.toCODECompare({
                    repoPath: repoMetadata?.path as string,
                    diffRefs: makeDiffRefs(repoMetadata?.defaultBranch as string, '')
                  })
                )
              }}
            />
          )
        }}>
        {/* TODO: Render pull request table here - https://www.figma.com/file/PgBvi804VdQNyLS8fD9K0p/SCM?node-id=1220%3A119902&t=D3DaDpST8oO95WSu-0  */}
      </PageBody>
    </Container>
  )
}
