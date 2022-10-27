import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useParams } from 'react-router-dom'
import { useGet } from 'restful-react'
import { useAppContext } from 'AppContext'
import type { SCMPathProps } from 'RouteDefinitions'
import type { TypesRepository } from 'services/scm'
import { getErrorMessage } from 'utils/Utils'
import { RepositoryContent } from './RepositoryContent/RepositoryContent'
import { RepositoryHeader } from './RepositoryHeader/RepositoryHeader'
import css from './Repository.module.scss'

export default function Repository(): JSX.Element {
  const { space: spaceFromParams, repoName, gitRef = '', resourcePath = '' } = useParams<SCMPathProps>()
  const { space = spaceFromParams || '' } = useAppContext()
  const { data, error, loading } = useGet<TypesRepository>({
    path: `/api/v1/repos/${space}/${repoName}/+/`
  })

  return (
    <Container className={css.main}>
      <PageBody loading={loading} error={error ? getErrorMessage(error) : null}>
        {data ? (
          <>
            <RepositoryHeader repoMetadata={data} />
            <RepositoryContent repoMetadata={data} gitRef={gitRef} resourcePath={resourcePath} />
          </>
        ) : null}
      </PageBody>
    </Container>
  )
}
