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
import classNames from 'classnames'
import type { IconName } from '@harnessio/icons'
import { FontVariation } from '@harnessio/design-system'
import { Button, ButtonVariation, Layout, Text } from '@harnessio/uicore'
import { useGetBulkDownloadRequestStatusQuery } from '@harnessio/react-har-service-v2-client'

import { useAppStore } from '@ar/hooks'
import { BulkDownloadRequestStatusEnum } from './types'
import css from './AsyncDownloadRequestsProvider.module.scss'

const POLL_INTERVAL = 3000

interface AsyncDownloadRequestStatusIconProps {
  status?: BulkDownloadRequestStatusEnum
  message?: string
}

function AsyncDownloadRequestStatusIcon(props: AsyncDownloadRequestStatusIconProps) {
  const { status, message } = props
  let icon: IconName = 'pending'
  switch (status) {
    case BulkDownloadRequestStatusEnum.PENDING:
      icon = 'pending'
      break
    case BulkDownloadRequestStatusEnum.PROCESSING:
      icon = 'steps-spinner'
      break
    case BulkDownloadRequestStatusEnum.SUCCESS:
      icon = 'success-tick'
      break
    case BulkDownloadRequestStatusEnum.FAILED:
      icon = 'solid-error'
      break
    default:
      icon = 'pending'
      break
  }
  return <Text icon={icon} iconProps={{ size: 20 }} tooltip={message} />
}

interface AsyncDownloadRequestProps {
  requestKey: string
  onRemove: (requestKey: string) => void
}
export default function AsyncDownloadRequest({ requestKey, onRemove }: AsyncDownloadRequestProps) {
  const { scope } = useAppStore()
  const { accountId } = scope
  const { data } = useGetBulkDownloadRequestStatusQuery(
    {
      queryParams: {
        account_identifier: accountId as string
      },
      download_id: requestKey
    },
    {
      refetchInterval: queryData => {
        const status = queryData?.content?.data?.status as BulkDownloadRequestStatusEnum | undefined
        // Stop polling if status is SUCCESS or FAILED
        if (
          [BulkDownloadRequestStatusEnum.SUCCESS, BulkDownloadRequestStatusEnum.FAILED].includes(
            status as BulkDownloadRequestStatusEnum
          )
        ) {
          return false
        }
        // Poll every 3 seconds for PENDING or PROCESSING status
        return POLL_INTERVAL
      }
    }
  )
  if (!data) return <></>
  const response = data.content.data
  return (
    <Layout.Horizontal className={classNames(css.requestContainer, css.cardContainer)}>
      <Layout.Horizontal spacing="small">
        <AsyncDownloadRequestStatusIcon
          status={response.status as BulkDownloadRequestStatusEnum}
          message={response.message}
        />
        <Text className={css.requestKey} lineClamp={1} font={{ variation: FontVariation.BODY, weight: 'bold' }}>
          {response.outputFileName}
        </Text>
      </Layout.Horizontal>
      <Layout.Horizontal spacing="small">
        {response.downloadUrl && (
          <Button
            href={
              typeof window.getApiBaseUrl === 'function'
                ? window.getApiBaseUrl(response.downloadUrl)
                : response.downloadUrl
            }
            target="_blank"
            variation={ButtonVariation.ICON}
            small
            icon="main-download"
            iconProps={{ size: 16 }}
            onClick={() => onRemove(requestKey)}
          />
        )}
        <Button
          variation={ButtonVariation.ICON}
          small
          icon="main-close"
          iconProps={{ size: 12 }}
          onClick={() => onRemove(requestKey)}
        />
      </Layout.Horizontal>
    </Layout.Horizontal>
  )
}
