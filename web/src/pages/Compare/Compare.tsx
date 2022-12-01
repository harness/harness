import React, { useState } from 'react'
import { Container, PageBody, NoDataCard } from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { getErrorMessage } from 'utils/Utils'
import emptyStateImage from 'images/empty-state.svg'
import { makeDiffRefs } from 'utils/GitUtils'
import { CompareContentHeader } from './CompareContentHeader/CompareContentHeader'
import css from './Compare.module.scss'

export default function Compare() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const { repoMetadata, error, loading, refetch, diffRefs } = useGetRepositoryMetadata()
  const [baseRef, setBaseRef] = useState(diffRefs.baseRef)
  const [compareRef, setCompareRef] = useState(diffRefs.compareRef)

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('comparingChanges')}
        dataTooltipId="comparingChanges"
      />
      <PageBody loading={loading} error={getErrorMessage(error)} retryOnError={() => refetch()}>
        {repoMetadata && (
          <CompareContentHeader
            repoMetadata={repoMetadata}
            baseRef={baseRef}
            compareRef={compareRef}
            onBaseRefChanged={gitRef => {
              setBaseRef(gitRef)
              history.replace(
                routes.toCODECompare({
                  repoPath: repoMetadata.path as string,
                  diffRefs: makeDiffRefs(gitRef, compareRef)
                })
              )
            }}
            onCompareRefChanged={gitRef => {
              setCompareRef(gitRef)
              history.replace(
                routes.toCODECompare({
                  repoPath: repoMetadata.path as string,
                  diffRefs: makeDiffRefs(baseRef, gitRef)
                })
              )
            }}
          />
        )}

        <Container className={css.noDataContainer}>
          <NoDataCard image={emptyStateImage} message={getString('selectToViewMore')} />
        </Container>
      </PageBody>
    </Container>
  )
}
