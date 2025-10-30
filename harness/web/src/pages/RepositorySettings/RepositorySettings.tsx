/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import cx from 'classnames'
import { PageBody, Container, Tabs } from '@harnessio/uicore'
import { useHistory } from 'react-router-dom'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useDisableCodeMainLinks } from 'hooks/useDisableCodeMainLinks'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import ProtectionRulesListing from 'components/ProtectionRules/ProtectionRulesListing'
import { useAppContext } from 'AppContext'
import { SettingsTab, normalizeGitRef } from 'utils/GitUtils'
import { getErrorMessage, voidFn } from 'utils/Utils'
import SecurityScanSettings from 'pages/RepositorySettings/SecurityScanSettings/SecurityScanSettings'
import LabelsListing from 'pages/Labels/LabelsListing'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import GeneralSettingsContent from './GeneralSettingsContent/GeneralSettingsContent'
import css from './RepositorySettings.module.scss'

export default function RepositorySettings() {
  const { repoMetadata, error, loading, refetch, settingSection, gitRef, resourcePath } = useGetRepositoryMetadata()
  const space = useGetSpaceParam()
  const history = useHistory()
  const { routes } = useAppContext()
  const [activeTab, setActiveTab] = React.useState<string>(settingSection || SettingsTab.GENERAL)
  const { getString } = useStrings()
  const { isRepositoryEmpty } = useGetResourceContent({
    repoMetadata,
    gitRef: normalizeGitRef(gitRef) as string,
    resourcePath
  })
  useDisableCodeMainLinks(!!isRepositoryEmpty)
  const tabListArray = [
    {
      id: SettingsTab.GENERAL,
      title: getString('settings'),
      panel: (
        <Container padding={'large'}>
          <GeneralSettingsContent
            repoMetadata={repoMetadata}
            refetch={refetch}
            gitRef={gitRef}
            isRepositoryEmpty={isRepositoryEmpty}
          />
        </Container>
      )
    },
    {
      id: SettingsTab.LABELS,
      title: getString('labels.labels'),
      panel: <LabelsListing activeTab={activeTab} repoMetadata={repoMetadata} space={space} />
    },
    {
      id: SettingsTab.PROTECTION_RULES,
      title: getString('protectionRules.title'),
      panel: <ProtectionRulesListing repoMetadata={repoMetadata} activeTab={activeTab} />
    },
    {
      id: SettingsTab.SECURITY,
      title: getString('security'),
      panel: <SecurityScanSettings repoMetadata={repoMetadata} activeTab={activeTab} />
    }
  ]
  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        className={css.headerContainer}
        repoMetadata={repoMetadata}
        title={getString('manageRepository')}
        dataTooltipId="repositorySettings"
      />
      <PageBody error={getErrorMessage(error)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading} />
        {repoMetadata && (
          <Container className={cx(css.main, css.tabsContainer)}>
            <Tabs
              id="SettingsTabs"
              large={false}
              defaultSelectedTabId={activeTab}
              animate={false}
              onChange={(id: string) => {
                setActiveTab(id)
                history.replace(
                  routes.toCODESettings({
                    repoPath: repoMetadata?.path as string,

                    settingSection: id !== SettingsTab.GENERAL ? (id as string) : ''
                  })
                )
              }}
              tabList={tabListArray}></Tabs>
          </Container>
        )}
      </PageBody>
    </Container>
  )
}
