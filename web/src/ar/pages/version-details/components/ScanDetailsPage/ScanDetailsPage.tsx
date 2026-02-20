/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useContext, useState } from 'react'
import { Container, Layout } from '@harnessio/uicore'

import { useParentComponents } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { ButtonTab, ButtonTabs } from '@ar/components/ButtonTabs/ButtonTabs'
import { VersionProviderContext } from '@ar/pages/version-details/context/VersionProvider'

import { ScanStatusDetailsTabEnum } from './types'

function ScanDetailsPage() {
  const [selectedTab, setSelectedTab] = useState<ScanStatusDetailsTabEnum>(ScanStatusDetailsTabEnum.Overview)
  const { data } = useContext(VersionProviderContext)
  const { purl = '' } = data || {}
  const { OssOverviewView, OssVulnerabilitiesView } = useParentComponents()
  const { getString } = useStrings()
  return (
    <Layout.Vertical padding="large" spacing="large">
      <ButtonTabs small bold selectedTabId={selectedTab} onChange={setSelectedTab}>
        <ButtonTab
          id={ScanStatusDetailsTabEnum.Overview}
          icon="document"
          iconProps={{ size: 12 }}
          panel={
            <Container>
              {OssOverviewView ? (
                <OssOverviewView
                  purl={purl}
                  onViewVulnerabilities={() => setSelectedTab(ScanStatusDetailsTabEnum.Vulnerabilities)}
                />
              ) : (
                getString('tabNotFound')
              )}
            </Container>
          }
          title={getString('versionDetails.securityTests.tabs.overview')}
        />
        <ButtonTab
          id={ScanStatusDetailsTabEnum.Vulnerabilities}
          icon="document"
          iconProps={{ size: 12 }}
          panel={
            <Container>
              {OssVulnerabilitiesView ? <OssVulnerabilitiesView purl={purl} /> : getString('tabNotFound')}
            </Container>
          }
          title={getString('versionDetails.securityTests.tabs.vulnerabilities')}
        />
      </ButtonTabs>
    </Layout.Vertical>
  )
}

export default ScanDetailsPage
