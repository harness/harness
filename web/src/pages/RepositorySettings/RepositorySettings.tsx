import React from 'react'
import { useHistory } from 'react-router-dom'

import { PageBody, Button, Intent, Container, Tabs } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { CodeIcon } from 'utils/GitUtils'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { getErrorMessage } from 'utils/Utils'
import emptyStateImage from 'images/empty-state.svg'
import hooks from './mockWebhooks.json'
import { SettingsContent } from './SettingsContent'
import css from './RepositorySettings.module.scss'

enum SettingsTab {
  webhooks = 'webhooks',
  general = 'general'
}
export default function RepositorySettings() {
  const { repoMetadata, error, loading } = useGetRepositoryMetadata()
  const { routes } = useAppContext()
  const history = useHistory()
  const [activeTab, setActiveTab] = React.useState<string>(SettingsTab.webhooks)

  const NewWebHookButton = (
    <Button
      type="button"
      text={'Create Webhook'}
      intent={Intent.PRIMARY}
      icon={CodeIcon.Add}
      className={css.btn}
      onClick={() => {
        history.push(
          routes.toCODECreateWebhook({
            repoPath: repoMetadata?.path as string
          })
        )
      }}
    />
  )
  const { getString } = useStrings()
  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('settings')}
        dataTooltipId="repositorySettings"
      />
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
              panel: <div> General content</div>,
              iconProps: { name: 'cog' }
            },
            {
              id: SettingsTab.webhooks,
              title: getString('webhooks'),
              iconProps: { name: 'code-webhook' },
              panel: (
                <PageBody
                  loading={loading}
                  error={getErrorMessage(error)}
                  className={css.webhooksContent}
                  noData={{
                    when: () => repoMetadata !== null,
                    message: getString('noWebHooks'),
                    image: emptyStateImage,
                    button: NewWebHookButton
                  }}>
                  <Container className={css.contentContainer}>
                    <Container>{NewWebHookButton}</Container>
                    {repoMetadata ? <SettingsContent repoMetadata={repoMetadata} hooks={hooks} /> : null}
                  </Container>
                </PageBody>
              )
            }
          ]}></Tabs>
      </Container>
    </Container>
  )
}
