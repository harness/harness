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
import { PageBody, Page, Layout } from '@harnessio/uicore'
import { Render } from 'react-jsx-match'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import LabelsListing from 'pages/Labels/LabelsListing'
import { useGetCurrentPageScope } from 'hooks/useGetCurrentPageScope'
import css from './ManageLabels.module.scss'

export default function ManageLabels() {
  const space = useGetSpaceParam()
  const { hooks } = useAppContext()
  const { CODE_PULLREQ_LABELS: isLabelEnabled } = hooks?.useFeatureFlags()
  const pageScope = useGetCurrentPageScope()
  const { getString } = useStrings()

  return (
    <Layout.Vertical className={css.main}>
      <Page.Header title={getString('labels.labels')} />
      <PageBody>
        <Render when={!!isLabelEnabled}>
          <LabelsListing currentPageScope={pageScope} space={space} />
        </Render>
      </PageBody>
    </Layout.Vertical>
  )
}
