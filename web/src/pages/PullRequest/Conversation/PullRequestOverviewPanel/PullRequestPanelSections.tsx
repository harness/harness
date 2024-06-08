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
import { Layout } from '@harnessio/uicore'
import { PanelSectionOutletPosition } from 'pages/PullRequest/PullRequestUtils'

interface PullRequestPanelSectionsProps {
  outlets?: Partial<Record<PanelSectionOutletPosition, React.ReactNode>>
}

const PullRequestPanelSections = (props: PullRequestPanelSectionsProps) => {
  const { outlets = {} } = props
  return (
    <Layout.Vertical>
      {outlets[PanelSectionOutletPosition.CHANGES]}
      {outlets[PanelSectionOutletPosition.COMMENTS]}
      {outlets[PanelSectionOutletPosition.CHECKS]}
      {outlets[PanelSectionOutletPosition.MERGEABILITY]}
    </Layout.Vertical>
  )
}

export default PullRequestPanelSections
