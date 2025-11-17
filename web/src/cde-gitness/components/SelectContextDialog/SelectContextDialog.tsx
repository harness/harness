/*
 * Copyright 2024 Harness, Inc.
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

import React, { useEffect, useMemo, useState } from 'react'
import { Dialog } from '@blueprintjs/core'
import { Button, ButtonVariation, Container, Layout, Text, Pagination } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import { Cloud } from 'iconoir-react'
import cx from 'classnames'
import { useStrings } from 'framework/strings'
import { type TypesGitspaceConfig, useListGitspaces } from 'services/cde'
import type { EnumGitspaceFilterState } from 'services/cde'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { getIconByRepoType, getRepoNameFromURL } from 'cde-gitness/utils/SelectRepository.utils'
import getProviderIcon from 'cde-gitness/utils/InfraProvider.utils'
import { GitspaceOwnerType } from 'cde-gitness/constants'
import codeSandboxLogo from 'cde-gitness/assests/codeSandboxLogo.svg?url'
import ResourceDetails from '../ResourceDetails/ResourceDetails'
import css from './SelectContextDialog.module.scss'

export interface SelectContextDialogProps {
  isOpen: boolean
  onClose: () => void
  onApply: (selected?: TypesGitspaceConfig) => void
  title?: string
  width?: number
  children?: React.ReactNode
  selectedGitspaceId?: string
}
const SelectContextDialog: React.FC<SelectContextDialogProps> = ({
  isOpen,
  onClose,
  onApply,
  title = 'Select Context',
  width = 1000,
  selectedGitspaceId
}) => {
  const [query, setQuery] = useState('')
  const RUNNING: EnumGitspaceFilterState = 'running'
  const [selectedId, setSelectedId] = useState<string | undefined>(undefined)
  const { getString } = useStrings()
  const [page, setPage] = useState(1)
  const [limit] = useState(10)
  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '' } = useGetCDEAPIParams()
  const {
    data: gitspaces = [],
    loading,
    refetch,
    response
  } = useListGitspaces({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
    queryParams: {
      page,
      limit,
      query: undefined,
      gitspace_owner: GitspaceOwnerType.SELF,
      gitspace_states: [RUNNING]
    },
    queryParamStringifyOptions: {
      arrayFormat: 'repeat'
    },
    lazy: true
  })

  useEffect(() => {
    if (!isOpen) return
    const handle = setTimeout(() => {
      refetch({
        queryParams: {
          page,
          limit,
          query: query || undefined,
          gitspace_owner: GitspaceOwnerType.SELF,
          gitspace_states: [RUNNING]
        },
        queryParamStringifyOptions: {
          arrayFormat: 'repeat'
        }
      })
    }, 300)
    return () => clearTimeout(handle)
  }, [isOpen, query, page, limit, refetch])

  useEffect(() => {
    setPage(1)
  }, [query])

  useEffect(() => {
    if (isOpen && selectedGitspaceId) {
      setSelectedId(selectedGitspaceId)
    }
  }, [isOpen, selectedGitspaceId])

  const totalItems = useMemo(() => parseInt(response?.headers?.get('x-total') || '0'), [response])
  const totalPages = useMemo(() => parseInt(response?.headers?.get('x-total-pages') || '0'), [response])
  const pageSize = useMemo(() => parseInt(response?.headers?.get('x-per-page') || String(limit)), [response, limit])
  const onSelect = (gs: TypesGitspaceConfig) => {
    setSelectedId(gs.identifier || '')
  }
  const renderCard = (gs: TypesGitspaceConfig) => {
    const isSelected = selectedId === gs.identifier
    const repoName = getRepoNameFromURL(gs.code_repo_url || '') || ''
    const branch = gs.branch || ''
    const cpu = gs.resource?.cpu ? String(gs.resource.cpu).trim() : undefined
    const memory = gs.resource?.memory ? String(gs.resource.memory).trim() : undefined
    const disk = gs.resource?.disk ? String(gs.resource.disk).trim() : undefined
    const md = (gs.resource?.metadata as Record<string, string> | null | undefined) || undefined
    const cores = md?.['cores']?.toString().trim() || md?.['cpu_cores']?.toString().trim()

    const specParts: string[] = []
    if (cpu) {
      const cpuLabel = /cpu/i.test(cpu) ? cpu : `${cpu} CPU`
      specParts.push(cpuLabel)
    }
    if (cores) {
      const coresLabel = /core/i.test(cores) ? cores : `${cores} cores`
      specParts.push(coresLabel)
    }
    if (memory) {
      const memHasUnit = /[a-zA-Z]/.test(memory)
      const memoryValue = memHasUnit ? memory : `${memory}GB`
      specParts.push(`${memoryValue} memory`)
    }
    if (disk) {
      const diskHasUnit = /[a-zA-Z]/.test(disk)
      const diskValue = diskHasUnit ? disk : `${disk}GB`
      specParts.push(`${diskValue} disk size`)
    }
    const specs = specParts.length ? `(${specParts.join(', ')})` : undefined
    return (
      <Container
        key={gs.identifier}
        padding="medium"
        onClick={() => onSelect(gs)}
        className={cx(css.card, { [css.selectedCardBorder]: isSelected })}>
        {isSelected && (
          <Container className={css.selectedCard}>
            <Icon name="tick" style={{ color: 'white' }} />
          </Container>
        )}
        <Layout.Vertical spacing="xsmall">
          <Text color={Color.BLACK} font={{ variation: FontVariation.BODY, weight: 'semi-bold' }} lineClamp={1}>
            {gs.identifier || gs.name}
          </Text>
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'start' }}>
            <Layout.Horizontal
              spacing="small"
              flex={{ alignItems: 'center', justifyContent: 'start' }}
              onClick={e => {
                e.preventDefault()
                e.stopPropagation()
                if (gs.code_repo_url) window.open(gs.code_repo_url, '_blank')
              }}>
              <Container height={14} width={14}>
                {getIconByRepoType({ repoType: gs.code_repo_type, height: 14 })}
              </Container>
              <Text lineClamp={1} color={Color.PRIMARY_7} title={gs.name} font={{ variation: FontVariation.BODY }}>
                {repoName}
              </Text>
            </Layout.Horizontal>
            <Layout.Horizontal
              spacing="small"
              flex={{ alignItems: 'center', justifyContent: 'start' }}
              onClick={e => {
                e.preventDefault()
                e.stopPropagation()
                if (gs.branch_url) window.open(gs.branch_url, '_blank')
              }}>
              <Text color={Color.PRIMARY_7}>:</Text>
              <Text
                lineClamp={1}
                icon="git-branch"
                iconProps={{ size: 12 }}
                color={Color.PRIMARY_7}
                title={branch}
                font={{ variation: FontVariation.BODY }}>
                {branch}
              </Text>
            </Layout.Horizontal>
          </Layout.Horizontal>
          {gs.resource && (
            <Layout.Horizontal spacing="xsmall">
              <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'start' }}>
                {getProviderIcon(gs.resource.infra_provider_type || '') ? (
                  <img
                    src={getProviderIcon(gs.resource.infra_provider_type || '') || undefined}
                    height={16}
                    width={16}
                    alt={'provider icon'}
                  />
                ) : (
                  <Cloud height={16} width={16} />
                )}
                <Text
                  lineClamp={1}
                  color={Color.GREY_500}
                  font={{ variation: FontVariation.BODY }}
                  title={gs.resource.config_name || gs.resource.config_identifier}>
                  {gs.resource.config_name || gs.resource.config_identifier}
                </Text>
              </Layout.Horizontal>
              <ResourceDetails resource={gs.resource} />
              {specs && (
                <Layout.Horizontal spacing="xsmall" flex={{ alignItems: 'center' }}>
                  <Text font={{ size: 'small' }} color={Color.GREY_600} lineClamp={1} title={specs}>
                    {specs}
                  </Text>
                </Layout.Horizontal>
              )}
            </Layout.Horizontal>
          )}
        </Layout.Vertical>
      </Container>
    )
  }

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      canEscapeKeyClose
      canOutsideClickClose
      style={{ width }}
      className={css.dialogContainer}>
      <Layout.Horizontal
        spacing="small"
        flex={{ justifyContent: 'space-between', alignItems: 'center' }}
        className={css.contextTitle}>
        <Text font={{ variation: FontVariation.H5 }}>{title}</Text>
        <Button aria-label="Close" icon="cross" variation={ButtonVariation.ICON} onClick={onClose} />
      </Layout.Horizontal>
      <Container className={css.contextBody}>
        <Layout.Vertical spacing="large">
          <Layout.Vertical spacing="small">
            <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
              <img src={codeSandboxLogo} height={20} width={20} alt="CodeSandbox" />
              <Text font={{ variation: FontVariation.BODY2, weight: 'semi-bold' }} color={Color.GREY_800}>
                {getString('cde.aiTasks.create.startWithActiveGitspace')}
              </Text>
            </Layout.Horizontal>
            <SearchInputWithSpinner
              query={query}
              setQuery={setQuery}
              loading={loading}
              placeholder={getString('cde.usageDashboard.searchPlaceholder')}
              spinnerPosition="right"
              fullWidth={true}
            />
          </Layout.Vertical>
          <Layout.Vertical spacing="small">
            {loading && (
              <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }}>
                <Text color={Color.GREY_600}>
                  Loading gitspaces <Icon name="loading" />
                </Text>
              </Layout.Horizontal>
            )}
            {!loading && (gitspaces?.length || 0) === 0 && (
              <Layout.Horizontal
                spacing="small"
                flex={{ justifyContent: 'center', alignItems: 'center' }}
                className={css.noGitspacesFound}>
                <Text color={Color.GREY_600}>{getString('cde.aiTasks.create.noGitspacesFound')}</Text>
              </Layout.Horizontal>
            )}
            {!loading && (gitspaces?.length || 0) > 0 && (
              <>
                <Container className={css.listScroll}>
                  {gitspaces?.map(gs => (
                    <Container key={gs.identifier} className={css.renderCardContainer}>
                      {renderCard(gs)}
                    </Container>
                  ))}
                </Container>
                {totalPages > 1 && (
                  <Layout.Horizontal
                    spacing="small"
                    flex={{ justifyContent: 'center' }}
                    className={css.paginationContainer}>
                    <Pagination
                      itemCount={totalItems}
                      pageCount={totalPages}
                      pageIndex={page - 1}
                      pageSize={pageSize}
                      gotoPage={index => setPage(index + 1)}
                    />
                  </Layout.Horizontal>
                )}
              </>
            )}
          </Layout.Vertical>
        </Layout.Vertical>
      </Container>
      <Layout.Horizontal
        spacing="small"
        flex={{ justifyContent: 'space-between', alignItems: 'center' }}
        className={css.contextFooter}>
        <Button text={getString('cde.settings.images.cancel')} variation={ButtonVariation.TERTIARY} onClick={onClose} />
        <Button
          text={getString('cde.settings.images.apply')}
          variation={ButtonVariation.PRIMARY}
          disabled={!selectedId}
          onClick={() => {
            const selected = (gitspaces || []).find(gs => gs.identifier === selectedId)
            onApply(selected)
          }}
        />
      </Layout.Horizontal>
    </Dialog>
  )
}

export default SelectContextDialog
