import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { voidFn, getErrorMessage } from 'utils/Utils'
import { RepositoryTagsContent } from './RepositoryTagsContent/RepositoryTagsContent'
import css from './RepositoryTags.module.scss'

export default function RepositoryTags() {
  const { getString } = useStrings()
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()

  return (
    <Container className={css.main}>
      <RepositoryPageHeader repoMetadata={repoMetadata} title={getString('tags')} dataTooltipId="repositoryTags" />
      <PageBody error={getErrorMessage(error)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading} />

        {repoMetadata ? <RepositoryTagsContent repoMetadata={repoMetadata} /> : null}
      </PageBody>
    </Container>
  )
}
