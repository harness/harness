import React from 'react'

import { PageBody, Container } from '@harnessio/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'

import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { getErrorMessage, voidFn } from 'utils/Utils'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import GeneralSettingsContent from './GeneralSettingsContent/GeneralSettingsContent'
import css from './RepositorySettings.module.scss'

export default function RepositorySettings() {
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()

  const { getString } = useStrings()
  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('settings')}
        dataTooltipId="repositorySettings"
      />
      <PageBody error={getErrorMessage(error)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading} />
        {repoMetadata && (
          <Container className={css.main} padding={'large'}>
            <GeneralSettingsContent repoMetadata={repoMetadata} refetch={refetch} />
          </Container>
        )}
      </PageBody>
    </Container>
  )
}
