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

import React, { useEffect, useState } from 'react'
import { Breadcrumbs, Layout, Page, useToaster, Text } from '@harnessio/uicore'
import { useParams } from 'react-router-dom'
import { Color, FontVariation } from '@harnessio/design-system'
import { GitspaceDetails } from 'cde/components/GitspaceDetails/GitspaceDetails'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { GitspaceLogs } from 'cde/components/GitspaceLogs/GitspaceLogs'
import { useStrings } from 'framework/strings'
import { CDEPathParams, useGetCDEAPIParams } from 'cde/hooks/useGetCDEAPIParams'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { useGetGitspace, useGetGitspaceInstanceLogs, useGitspaceAction } from 'services/cde'
import { getErrorMessage } from 'utils/Utils'
import { GitspaceActionType, GitspaceStatus } from 'cde/constants'
import Gitspace from '../../icons/Gitspace.svg?url'
import css from './GitspaceDetail.module.scss'

const GitspaceDetail = () => {
  const { showError } = useToaster()
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { routes } = useAppContext()
  const { gitspaceId = '' } = useParams<{ gitspaceId?: string }>()
  const { updateQueryParams } = useUpdateQueryParams<{ redirectFrom?: string }>()
  const { accountIdentifier, orgIdentifier, projectIdentifier } = useGetCDEAPIParams() as CDEPathParams
  const { redirectFrom = '' } = useQueryParams<{ redirectFrom?: string }>()
  const [startTriggred, setStartTriggred] = useState(false)

  const { data, loading, error, refetch } = useGetGitspace({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
    gitspaceIdentifier: gitspaceId || ''
  })

  const {
    data: logsData,
    loading: logsLoading,
    error: logsError,
    refetch: refetchLogs
  } = useGetGitspaceInstanceLogs({
    lazy: !!redirectFrom,
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
    gitspaceIdentifier: gitspaceId
  })

  const { state } = data || {}

  const {
    mutate,
    loading: mutateLoading,
    error: startError
  } = useGitspaceAction({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
    gitspaceIdentifier: gitspaceId || ''
  })

  useEffect(() => {
    const startTrigger = async () => {
      if (redirectFrom && !startTriggred && !mutateLoading) {
        try {
          setStartTriggred(true)
          await mutate({ action: GitspaceActionType.START })
          await refetch()
          await refetchLogs()
          updateQueryParams({ redirectFrom: undefined })
        } catch (err) {
          showError(getErrorMessage(err))
        }
      }
    }

    if (state && state !== GitspaceStatus.RUNNING && redirectFrom) {
      startTrigger()
    }
  }, [state, redirectFrom, mutateLoading, startTriggred])

  return (
    <>
      <Page.Header
        title={
          <Layout.Vertical>
            <Text font={{ variation: FontVariation.CARD_TITLE }}>
              {` ${getString('cde.gitspaceDetail')} ${data?.config?.name}`}
            </Text>
            <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
              {data?.config?.id}
            </Text>
          </Layout.Vertical>
        }
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
                  label: getString('cde.gitspaceDetail')
                }
              ]}
            />
          </Layout.Horizontal>
        }
      />
      <Page.Body
        loading={loading}
        loadingMessage="Fetching Gitspace Details ...."
        error={getErrorMessage(error)}
        retryOnError={() => refetch()}>
        <Layout.Horizontal className={css.main} spacing="medium">
          <GitspaceDetails
            data={data}
            error={error}
            loading={loading}
            refetch={refetch}
            refetchLogs={refetchLogs}
            mutate={mutate}
            actionError={startError}
            mutateLoading={mutateLoading}
          />
          <GitspaceLogs data={logsData} refetch={refetchLogs} loading={logsLoading} error={logsError} />
        </Layout.Horizontal>
      </Page.Body>
    </>
  )
}

export default GitspaceDetail
