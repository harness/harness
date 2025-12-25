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

import { Color } from '@harnessio/design-system'
import { Button, ButtonVariation, Container, Layout, Text } from '@harnessio/uicore'
import React from 'react'
import ReactTimeago from 'react-timeago'
import { PopoverPosition } from '@blueprintjs/core'
import { Circle, InfoEmpty, Cloud } from 'iconoir-react'
import type { IconName } from '@harnessio/icons'
import Secret from 'cde-gitness/assests/secret.svg?url'
import { useStrings } from 'framework/strings'
import { GitspaceStatus } from 'cde-gitness/constants'
import { getGitspaceChanges, getIconByRepoType } from 'cde-gitness/utils/SelectRepository.utils'
import type { TypesGitspaceConfig } from 'services/cde'
import { getStatusColor, getStatusText } from '../GitspaceListing/ListGitspaces'
import getProviderIcon from '../../utils/InfraProvider.utils'
import ResourceDetails from '../ResourceDetails/ResourceDetails'
import { getRepoNameFromURL } from '../../utils/SelectRepository.utils'
import css from './DetailsCard.module.scss'

export const DetailsCard = ({
  data,
  standalone
}: {
  data: TypesGitspaceConfig | null
  standalone?: boolean
  loading?: boolean
}) => {
  const { getString } = useStrings()
  const { branch, state, name, branch_url, code_repo_url, code_repo_type, instance, resource } = data || {}
  const repoName = getRepoNameFromURL(code_repo_url) || ''

  const { has_git_changes } = instance || {}
  const gitChanges = getGitspaceChanges(has_git_changes, getString, '--')
  const color = getStatusColor(state)
  const customProps =
    state === GitspaceStatus.STARTING
      ? {
          icon: 'loading' as IconName,
          iconProps: { color: Color.PRIMARY_4 }
        }
      : { icon: undefined }
  return (
    <>
      <Layout.Horizontal width={'90%'} className={css.detailsContainer} padding={{ bottom: 'xlarge', top: 'xlarge' }}>
        <Layout.Vertical
          spacing="small"
          flex={{ justifyContent: 'center', alignItems: 'flex-start' }}
          className={css.marginLeftContainer}>
          <Text className={css.rowHeaders}>{getString('cde.status')}</Text>
          <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
            {state !== GitspaceStatus.STARTING && <Circle height={10} width={10} color={color} fill={color} />}
            <Text {...customProps} className={css.statusText} title={name}>
              {getStatusText(getString, state)}
            </Text>
          </Layout.Horizontal>
        </Layout.Vertical>
        <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
          <Text className={css.rowHeaders}>{getString('cde.repositoryAndBranch')}</Text>
          <Layout.Horizontal
            spacing="small"
            flex={{ alignItems: 'center', justifyContent: 'start' }}
            onClick={e => {
              e.preventDefault()
              e.stopPropagation()
            }}>
            {getIconByRepoType({ repoType: code_repo_type, height: 20 })}
            <Text title={'RepoName'} className={css.clickableText} onClick={() => window.open(code_repo_url, '_blank')}>
              {repoName}
            </Text>
            <Text color={Color.PRIMARY_7}>:</Text>
            <Text
              iconProps={{ size: 10 }}
              icon="git-branch"
              className={css.clickableText}
              onClick={() => window.open(branch_url, '_blank')}>
              {branch}
            </Text>
          </Layout.Horizontal>
        </Layout.Vertical>
        {!standalone && (
          <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
            <Text className={css.rowHeaders}>{getString('cde.infraProvider')}</Text>
            <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
              {(() => {
                const providerType = resource?.infra_provider_type || ''
                const providerConfigId = resource?.config_identifier || ''
                const providerIcon = getProviderIcon(providerType)

                const displayName = resource?.config_name || providerConfigId

                return (
                  <>
                    {providerIcon ? (
                      <img src={providerIcon} className={css.standardIcon} alt={'provider icon'} />
                    ) : (
                      <Cloud className={css.standardIcon} />
                    )}
                    <Text lineClamp={1} className={css.providerText} title={displayName}>
                      {displayName}
                    </Text>
                  </>
                )
              })()}
            </Layout.Horizontal>
          </Layout.Vertical>
        )}
        {!standalone && (
          <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
            <Text className={css.rowHeaders}>{getString('cde.regionMachineType')}</Text>
            <ResourceDetails resource={resource} />
          </Layout.Vertical>
        )}
        {/* Conditional SSH Key field - only shown when ssh_token_identifier exists */}
        {data?.ssh_token_identifier && (
          <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
            <Text className={css.rowHeaders}>{'SSH Key'}</Text>
            <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
              <img src={Secret} className={css.standardIcon} alt={'secret'} />
              <Text lineClamp={1} className={css.providerText} title={data.ssh_token_identifier}>
                {data.ssh_token_identifier}
              </Text>
            </Layout.Horizontal>
          </Layout.Vertical>
        )}
        <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
          <Layout.Horizontal
            flex={{ alignItems: 'center', justifyContent: 'start' }}
            className={css.horizontalContainer}>
            <Text className={css.rowHeaders}>{getString('cde.lastStarted')}</Text>
            <Button
              className={css.infoButton}
              variation={ButtonVariation.ICON}
              tooltip={
                <Container width={300} padding="medium">
                  <Layout.Vertical spacing="small">
                    <Text font="small" color={Color.WHITE}>
                      {getString('cde.lastStartedTooltip')}
                    </Text>
                  </Layout.Vertical>
                </Container>
              }
              tooltipProps={{ isDark: true, position: PopoverPosition.AUTO }}>
              <InfoEmpty className={css.infoIcon} />
            </Button>
          </Layout.Horizontal>
          {instance?.active_time_started ? (
            <ReactTimeago date={instance?.active_time_started || 0} />
          ) : (
            <Text className={css.greyText}>{getString('cde.na')}</Text>
          )}
        </Layout.Vertical>
        <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
          <Layout.Horizontal
            flex={{ alignItems: 'center', justifyContent: 'start' }}
            className={css.horizontalContainer}>
            <Text className={css.rowHeaders}>{getString('cde.lastUsed')}</Text>
            <Button
              className={css.infoButton}
              variation={ButtonVariation.ICON}
              tooltip={
                <Container width={300} padding="medium">
                  <Layout.Vertical spacing="small">
                    <Text font="small" color={Color.WHITE}>
                      {getString('cde.lastUsedTooltip')}
                    </Text>
                  </Layout.Vertical>
                </Container>
              }
              tooltipProps={{ isDark: true, position: PopoverPosition.AUTO }}>
              <InfoEmpty className={css.infoIcon} />
            </Button>
          </Layout.Horizontal>
          {instance?.last_used ? (
            <ReactTimeago date={instance?.last_used || 0} />
          ) : (
            <Text className={css.greyText}>{getString('cde.na')}</Text>
          )}
        </Layout.Vertical>
        <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
          <Layout.Horizontal
            flex={{ alignItems: 'center', justifyContent: 'start' }}
            className={css.horizontalContainer}>
            <Text className={css.rowHeaders}>{getString('cde.changes')}</Text>
            <Button
              className={css.infoButton}
              variation={ButtonVariation.ICON}
              tooltip={
                <Container width={300} padding="medium">
                  <Layout.Vertical spacing="small">
                    <Text font="small" color={Color.WHITE}>
                      {getString('cde.changesTooltip.description')}
                    </Text>
                  </Layout.Vertical>
                </Container>
              }
              tooltipProps={{ isDark: true, position: PopoverPosition.AUTO }}>
              <InfoEmpty className={css.infoIcon} />
            </Button>
          </Layout.Horizontal>
          <Text className={css.greyText}>{gitChanges}</Text>
        </Layout.Vertical>
      </Layout.Horizontal>
    </>
  )
}
