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

import React, { useState, useEffect } from 'react'
import cx from 'classnames'
import { Container, Tabs, Page } from '@harnessio/uicore'
import { omit } from 'lodash-es'
import { useStrings } from 'framework/strings'
import { PullRequestFilterOption, PullRequestReviewFilterOption, SpacePRTabs } from 'utils/GitUtils'
import { PageBrowserProps, ScopeLevelEnum } from 'utils/Utils'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { SpacePullRequestsListing } from 'pages/PullRequests/SpacePullRequestsListing'
import css from '../ManageSpace.module.scss'

export default function SpacePullRequests() {
  const browserParams = useQueryParams<PageBrowserProps>()
  const { updateQueryParams, replaceQueryParams } = useUpdateQueryParams()
  const [activeTab, setActiveTab] = useState<string>(browserParams.tab || SpacePRTabs.CREATED)
  const [includeSubspaces, setIncludeSubspaces] = useState<ScopeLevelEnum>(
    browserParams?.subspace || ScopeLevelEnum.CURRENT
  )
  const { getString } = useStrings()

  useEffect(() => {
    const params = {
      ...browserParams,
      tab: browserParams.tab ?? SpacePRTabs.CREATED,
      ...(!browserParams.state && { state: PullRequestFilterOption.OPEN })
    }
    updateQueryParams(params, undefined, true)
  }, [browserParams])

  const tabListArray = [
    {
      id: SpacePRTabs.CREATED,
      title: getString('pr.myPRs'),
      panel: (
        <SpacePullRequestsListing
          activeTab={SpacePRTabs.CREATED}
          includeSubspaces={includeSubspaces}
          setIncludeSubspaces={setIncludeSubspaces}
        />
      )
    },
    {
      id: SpacePRTabs.REVIEW_REQUESTED,
      title: getString('pr.reviewRequested'),
      panel: (
        <SpacePullRequestsListing
          activeTab={SpacePRTabs.REVIEW_REQUESTED}
          includeSubspaces={includeSubspaces}
          setIncludeSubspaces={setIncludeSubspaces}
        />
      )
    }
  ]
  return (
    <Container className={css.main}>
      <Page.Header title={getString('pullRequests')} />

      <Container className={cx(css.main, css.tabsContainer)}>
        <Tabs
          id="SettingsTabs"
          large={false}
          defaultSelectedTabId={activeTab}
          animate={false}
          onChange={(id: string) => {
            setActiveTab(id)
            const params = {
              ...browserParams,
              tab: id
            }
            if (id === SpacePRTabs.CREATED) {
              replaceQueryParams(omit(params, 'review'), undefined, true)
            } else {
              const updatedParams = { ...params, review: PullRequestReviewFilterOption.PENDING }
              replaceQueryParams(updatedParams, undefined, true)
            }
          }}
          tabList={tabListArray}></Tabs>
      </Container>
    </Container>
  )
}
