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
import { Container, Tabs, Page } from '@harnessio/uicore'
import { useHistory, useParams } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import BranchProtectionListing from 'components/BranchProtection/BranchProtectionListing'
import { SettingsTab } from 'utils/GitUtils'
import LabelsListing from 'pages/Labels/LabelsListing'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { CODEProps } from 'RouteDefinitions'
import css from '../ManageSpace.module.scss'

export default function ManageRepositories() {
  const { settingSection } = useParams<CODEProps>()
  const space = useGetSpaceParam()
  const history = useHistory()
  const { routes } = useAppContext()
  const [activeTab, setActiveTab] = React.useState<string>(settingSection || SettingsTab.labels)
  const { getString } = useStrings()

  const tabListArray = [
    {
      id: SettingsTab.labels,
      title: getString('labels.labels'),
      panel: <LabelsListing activeTab={activeTab} space={space} />
    },
    {
      id: SettingsTab.branchProtection,
      title: getString('branchProtection.title'),
      panel: <BranchProtectionListing activeTab={activeTab} />
    }
  ]
  return (
    <Container className={css.main}>
      <Page.Header title={getString('manageRepositories')} />

      <Container className={cx(css.main, css.tabsContainer)}>
        <Tabs
          id="SettingsTabs"
          large={false}
          defaultSelectedTabId={activeTab}
          animate={false}
          onChange={(id: string) => {
            setActiveTab(id)
            history.replace(
              routes.toCODEManageRepositories({
                space,
                settingSection: id !== SettingsTab.labels ? (id as string) : ''
              })
            )
          }}
          tabList={tabListArray}></Tabs>
      </Container>
    </Container>
  )
}
