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
import { PageBody, Container, Tabs, Page } from '@harnessio/uicore'
import { useHistory } from 'react-router-dom'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { SettingsTab } from 'utils/GitUtils'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import LabelsListing from 'pages/Labels/LabelsListing'
import ProtectionRulesListing from 'components/ProtectionRules/ProtectionRulesListing'
import GeneralSpaceSettings from './GeneralSettings/GeneralSpaceSettings'
import css from './SpaceSettings.module.scss'

export default function SpaceSettings() {
  const { settingSection } = useGetRepositoryMetadata()
  const history = useHistory()
  const { routes } = useAppContext()
  const space = useGetSpaceParam()
  const [activeTab, setActiveTab] = React.useState<string>(settingSection || SettingsTab.GENERAL)
  const { getString } = useStrings()

  const tabListArray = [
    {
      id: SettingsTab.GENERAL,
      title: 'General',
      panel: (
        <Container padding={'large'}>
          <GeneralSpaceSettings />
        </Container>
      )
    },
    {
      id: SettingsTab.LABELS,
      title: getString('labels.labels'),
      panel: <LabelsListing activeTab={activeTab} space={space} />
    },
    {
      id: SettingsTab.PROTECTION_RULES,
      title: getString('protectionRules.title'),
      panel: <ProtectionRulesListing activeTab={activeTab} />
    }
  ]
  return (
    <Container className={css.main}>
      <Page.Header title={getString('spaceSetting.settings')} />
      <PageBody>
        <Container className={cx(css.main, css.tabsContainer)}>
          <Tabs
            id="SpaceSettingsTabs"
            large={false}
            defaultSelectedTabId={activeTab}
            animate={false}
            onChange={(id: string) => {
              setActiveTab(id)
              history.replace(
                routes.toCODESpaceSettings({
                  space: space as string,
                  settingSection: id !== SettingsTab.GENERAL ? (id as string) : ''
                })
              )
            }}
            tabList={tabListArray}></Tabs>
        </Container>
      </PageBody>
    </Container>
  )
}
