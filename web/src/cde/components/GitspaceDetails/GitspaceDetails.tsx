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
import { Text, Layout, Container, Button, ButtonVariation, PageError } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import type { PopoverProps } from '@harnessio/uicore/dist/components/Popover/Popover'
import { Menu, MenuItem, PopoverPosition } from '@blueprintjs/core'
import { useParams } from 'react-router-dom'
import { Cpu, Circle, GitFork, Repository } from 'iconoir-react'
import { isUndefined } from 'lodash-es'
import { useGetGitspace, useGitspaceAction } from 'services/cde'
import { CDEPathParams, useGetCDEAPIParams } from 'cde/hooks/useGetCDEAPIParams'
import { useStrings } from 'framework/strings'
import { GitspaceActionType, GitspaceStatus, IDEType } from 'cde/constants'
import { getErrorMessage } from 'utils/Utils'
import Gitspace from '../../icons/Gitspace.svg?url'
import css from './GitspaceDetails.module.scss'

interface QueryGitspace {
  gitspaceId?: string
}

export const GitspaceDetails = () => {
  const { getString } = useStrings()
  const { accountIdentifier, orgIdentifier, projectIdentifier } = useGetCDEAPIParams() as CDEPathParams
  const { gitspaceId } = useParams<QueryGitspace>()

  const { data, loading, error, refetch } = useGetGitspace({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
    gitspaceIdentifier: gitspaceId || ''
  })

  const { config, status } = data || {}

  const { mutate } = useGitspaceAction({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
    gitspaceIdentifier: gitspaceId || ''
  })

  const openEditorLabel =
    config?.ide === IDEType.VSCODE ? getString('cde.details.openEditor') : getString('cde.details.openBrowser')

  return (
    <Layout.Vertical width={'30%'} spacing="large">
      <Layout.Vertical spacing="medium">
        <img src={Gitspace} width={42} height={42}></img>
        {error ? (
          <PageError onClick={() => refetch()} message={getErrorMessage(error)} />
        ) : (
          <Text className={css.subText} font={{ variation: FontVariation.CARD_TITLE }}>
            {status === GitspaceStatus.UNKNOWN && getString('cde.details.provisioningGitspace')}
            {status === GitspaceStatus.STOPPED && getString('cde.details.gitspaceStopped')}
            {status === GitspaceStatus.RUNNING && getString('cde.details.gitspaceRunning')}
            {loading && getString('cde.details.fetchingGitspace')}
            {!loading && isUndefined(status) && getString('cde.details.noData')}
          </Text>
        )}
      </Layout.Vertical>
      <Container className={css.detailsBar}>
        {error ? (
          <Text>{getErrorMessage(error)}</Text>
        ) : (
          <>
            {isUndefined(config) ? (
              <Text>{getString('cde.details.noData')}</Text>
            ) : (
              <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
                <Layout.Vertical spacing="small">
                  <Layout.Horizontal spacing="small">
                    <Circle color="#32CD32" fill="#32CD32" />
                    <Text font={'small'}>{config?.code_repo_id?.toUpperCase()}</Text>
                  </Layout.Horizontal>
                  <Layout.Horizontal spacing="small">
                    <Repository />
                    <Text font={'small'}>{config?.code_repo_id}</Text>
                    <Text> / </Text>
                    <GitFork />
                    <Text font={'small'}>{config?.branch}</Text>
                  </Layout.Horizontal>
                </Layout.Vertical>
                <Layout.Vertical spacing="small">
                  <Layout.Horizontal spacing="small">
                    <Cpu />
                    <Text font={'small'}>{config?.infra_provider_resource_id}</Text>
                  </Layout.Horizontal>
                </Layout.Vertical>
              </Layout.Horizontal>
            )}
          </>
        )}
      </Container>
      <Layout.Horizontal spacing={'medium'}>
        {status === GitspaceStatus.UNKNOWN && (
          <>
            <Button variation={ButtonVariation.SECONDARY} disabled>
              {openEditorLabel}
            </Button>
            <Button variation={ButtonVariation.PRIMARY} onClick={() => mutate({ action: 'STOP' })}>
              {getString('cde.details.stopProvising')}
            </Button>
          </>
        )}

        {status === GitspaceStatus.STOPPED && (
          <>
            <Button variation={ButtonVariation.PRIMARY} onClick={() => mutate({ action: GitspaceActionType.START })}>
              {getString('cde.details.startGitspace')}
            </Button>
            <Button variation={ButtonVariation.TERTIARY}>{getString('cde.details.goToDashboard')}</Button>
          </>
        )}

        {status === GitspaceStatus.RUNNING && (
          <>
            <Button variation={ButtonVariation.PRIMARY}>{openEditorLabel}</Button>
            <Button
              variation={ButtonVariation.TERTIARY}
              rightIcon="chevron-down"
              tooltipProps={
                {
                  interactionKind: 'click',
                  position: PopoverPosition.BOTTOM_LEFT,
                  popoverClassName: css.popover
                } as PopoverProps
              }
              tooltip={
                <Menu>
                  <MenuItem
                    text={getString('cde.details.stopGitspace')}
                    onClick={() => mutate({ action: GitspaceActionType.STOP })}
                  />
                </Menu>
              }>
              {getString('cde.details.actions')}
            </Button>
          </>
        )}
      </Layout.Horizontal>
    </Layout.Vertical>
  )
}
