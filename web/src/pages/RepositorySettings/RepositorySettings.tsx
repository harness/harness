import React from 'react'

import { PageBody, Container, Tabs } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'

import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { getErrorMessage, voidFn } from 'utils/Utils'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import GeneralSettingsContent from './GeneralSettingsContent/GeneralSettingsContent'
import css from './RepositorySettings.module.scss'

enum SettingsTab {
  webhooks = 'webhooks',
  general = 'general'
}
export default function RepositorySettings() {
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()

  const [activeTab, setActiveTab] = React.useState<string>(SettingsTab.general)

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
            <Tabs
              id="SettingsTabs"
              vertical
              large={false}
              defaultSelectedTabId={activeTab}
              animate={false}
              onChange={(id: string) => setActiveTab(id)}
              tabList={[
                {
                  id: SettingsTab.general,
                  title: getString('general'),
                  panel: <GeneralSettingsContent repoMetadata={repoMetadata} refetch={refetch} />,
                  iconProps: { name: 'cog' }
                }
              ]}></Tabs>
          </Container>
        )}
      </PageBody>
    </Container>
  )
}
