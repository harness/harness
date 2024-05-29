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
import { Breadcrumbs, Layout, Page } from '@harnessio/uicore'
import { useParams } from 'react-router-dom'
import { GitspaceDetails } from 'cde/components/GitspaceDetails/GitspaceDetails'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { GitspaceLogs } from 'cde/components/GitspaceLogs/GitspaceLogs'
import { useStrings } from 'framework/strings'
import Gitspace from '../../icons/Gitspace.svg?url'
import css from './GitspaceDetail.module.scss'

const GitspaceDetail = () => {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { routes } = useAppContext()
  const { gitspaceId = '' } = useParams<{ gitspaceId?: string }>()

  return (
    <>
      <Page.Header
        title=""
        breadcrumbs={
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }}>
            <img src={Gitspace} height={20} width={20} style={{ marginRight: '5px' }} />
            <Breadcrumbs
              links={[
                {
                  url: routes.toCDEGitspaces({ space }),
                  label: getString('cde.cloudDeveloperExperience')
                },
                {
                  url: routes.toCDEGitspaceDetail({ space, gitspaceId }),
                  label: `${getString('cde.gitpsaceDetail')} ${gitspaceId}`
                }
              ]}
            />
          </Layout.Horizontal>
        }
      />
      <Page.Body>
        <Layout.Horizontal className={css.main} spacing="medium">
          <GitspaceDetails />
          <GitspaceLogs />
        </Layout.Horizontal>
      </Page.Body>
    </>
  )
}

export default GitspaceDetail
