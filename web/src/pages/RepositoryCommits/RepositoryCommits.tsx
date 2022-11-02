import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useParams } from 'react-router-dom'
import { useGet } from 'restful-react'
import { useAppContext } from 'AppContext'
import type { SCMPathProps } from 'RouteDefinitions'
import type { TypesRepository } from 'services/scm'
import { getErrorMessage } from 'utils/Utils'
import { RepositoryCommitsContent } from './RepositoryCommitsContent/RepositoryCommitsContent'
import { RepositoryCommitsHeader } from './RepositoryCommitsHeader/RepositoryCommitsHeader'
import css from './RepositoryCommits.module.scss'

export default function RepositoryCommits(): JSX.Element {
  const { space: spaceFromParams, repoName, commitRef } = useParams<SCMPathProps>()
  const { space = spaceFromParams || '' } = useAppContext()
  const { data, error, loading } = useGet<TypesRepository>({
    path: `/api/v1/repos/${space}/${repoName}/+/`
  })

  return (
    <Container className={css.main}>
      <PageBody loading={loading} error={error ? getErrorMessage(error) : null}>
        {data ? (
          <>
            <RepositoryCommitsHeader repoMetadata={data} />
            <RepositoryCommitsContent repoMetadata={data} commitRef={commitRef || (data.defaultBranch as string)} />
          </>
        ) : null}
      </PageBody>
    </Container>
  )
}
