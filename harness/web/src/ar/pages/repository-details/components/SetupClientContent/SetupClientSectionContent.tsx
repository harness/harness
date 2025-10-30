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

import React from 'react'
import { Container, Tab, Tabs } from '@harnessio/uicore'
import type { ClientSetupSection, TabSetupStepConfig } from '@harnessio/react-har-service-client'

import InlineSectionContent from './InlineSectionContent'
import css from './SetupClientContent.module.scss'

interface SetupClientSectionContentProps {
  section: ClientSetupSection
}

export default function SetupClientSectionContent(props: SetupClientSectionContentProps) {
  const { section } = props
  switch (section.type) {
    case 'TABS':
      return <TabsSectionContent section={section} />
    case 'INLINE':
    default:
      return <InlineSectionContent section={section} />
  }
}

interface TabsSectionContentProps {
  section: ClientSetupSection & TabSetupStepConfig
}

function TabsSectionContent(props: TabsSectionContentProps): JSX.Element {
  const { section } = props
  return (
    <Container className={css.tabsContainer}>
      <Tabs id="setup-client-tabs" defaultSelectedTabId={section.tabs && section.tabs[0].header}>
        {section.tabs?.map((tab, index) => (
          <Tab
            key={index}
            id={tab.header}
            title={tab.header}
            panel={
              <Container>
                {tab.sections?.map((each, idx) => (
                  <SetupClientSectionContent key={idx} section={each} />
                ))}
              </Container>
            }
          />
        ))}
      </Tabs>
    </Container>
  )
}
