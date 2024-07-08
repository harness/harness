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

import React, { useCallback, useState } from 'react'
import { Button, ButtonVariation, Card, Container, Layout, Text, TextInput } from '@harnessio/uicore'
import { Menu, MenuItem } from '@blueprintjs/core'
import { Color, FontVariation } from '@harnessio/design-system'
import { Link } from 'react-router-dom'
import { debounce, get } from 'lodash-es'
import { Icon } from '@harnessio/icons'
import { useFormikContext } from 'formik'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { OpenapiCreateGitspaceRequest, OpenapiGetCodeRepositoryResponse, useGetCodeRepository } from 'services/cde'
import { GitspaceSelect } from 'cde/components/GitspaceSelect/GitspaceSelect'
import { useStrings } from 'framework/strings'
import { CodeRepoAccessType } from 'cde/constants'
import { getIconByRepoType, getRepoIdFromURL, getRepoNameFromURL, isValidUrl } from './SelectRepository.utils'
import css from './SelectRepository.module.scss'

const RepositoryText = ({ repoURL }: { repoURL?: string }) => {
  const { getString } = useStrings()
  const repoName = getRepoNameFromURL(repoURL)
  return repoURL ? (
    <Layout.Vertical spacing="xsmall">
      <Text font={'normal'}>{repoName || getString('cde.repository.repo')}</Text>
      <Text font={'small'} lineClamp={1}>
        {repoURL || getString('cde.repository.repositoryURL')}
      </Text>
    </Layout.Vertical>
  ) : (
    <Text font={'normal'}>{getString('cde.repository.selectRepository')}</Text>
  )
}

const SelectRepositoryCard = ({
  data,
  onChange,
  resetData
}: {
  data: OpenapiGetCodeRepositoryResponse
  onChange: (formdata: any) => void
  resetData?: React.Dispatch<React.SetStateAction<OpenapiGetCodeRepositoryResponse | undefined>>
}) => {
  const { getString } = useStrings()

  return data ? (
    <Card className={css.repometadataCard}>
      {data?.access_type === CodeRepoAccessType.PRIVATE ? (
        <Layout.Vertical
          className={css.privateCard}
          flex={{ justifyContent: 'center', alignItems: 'center' }}
          spacing="medium">
          <Icon name="lock" size={20} />
          <Text>{getString('cde.repository.privateRepoWarning')}</Text>
          <Button variation={ButtonVariation.PRIMARY}>{`${getString('cde.repository.continueWith')} ${
            data?.repo_type
          }`}</Button>
        </Layout.Vertical>
      ) : (
        <Menu className={css.repometadata}>
          <MenuItem
            className={css.metadataItem}
            onClick={() => {
              onChange((prv: any) => {
                const repoId = getRepoIdFromURL(data?.url)
                return {
                  ...prv,
                  values: {
                    ...prv.values,
                    id: `${repoId}`,
                    name: getRepoNameFromURL(data?.url),
                    code_repo_url: data?.url || '',
                    code_repo_type: data?.repo_type || '',
                    code_repo_id: repoId,
                    branch: data?.branch
                  }
                }
              })
              resetData?.(undefined)
            }}
            text={
              <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                {getIconByRepoType({ repoType: data?.repo_type })}
                <Text className={css.repoDetail} lineClamp={1}>
                  {getRepoNameFromURL(data?.url)}
                </Text>
                <Text className={css.publicTag} font={'xsmall'}>
                  {data?.access_type}
                </Text>
              </Layout.Horizontal>
            }
          />
        </Menu>
      )}
    </Card>
  ) : (
    <></>
  )
}

export const SelectRepository = ({ disabled }: { disabled?: boolean }) => {
  const { getString } = useStrings()
  const { values, errors, setFormikState } = useFormikContext<OpenapiCreateGitspaceRequest>()

  const { code_repo_url } = values
  const space = useGetSpaceParam()

  const [error, setError] = useState<string>()
  const [repoMetadata, setRepoMetadata] = useState<OpenapiGetCodeRepositoryResponse | undefined>()

  const { mutate, loading } = useGetCodeRepository({
    accountIdentifier: space?.split('/')[0],
    orgIdentifier: space?.split('/')[1],
    projectIdentifier: space?.split('/')[2]
  })

  const onChange = useCallback(
    debounce(async (url: string) => {
      let errorMessage = ''
      try {
        if (isValidUrl(url)) {
          const response = await mutate({ url })
          if (response?.access_type === CodeRepoAccessType.PRIVATE) {
            errorMessage = getString('cde.repository.privateRepoWarning')
          }
          setRepoMetadata(response)
        } else {
          errorMessage = 'Invalid URL Format'
        }
      } catch (err) {
        errorMessage = get(err, 'message') || ''
      }
      setError(errorMessage)
    }, 1000),
    []
  )

  return (
    <Container width={'63%'}>
      <GitspaceSelect
        text={<RepositoryText repoURL={code_repo_url} />}
        icon={'code'}
        errorMessage={errors.code_repo_url}
        formikName="code_repo_url"
        tooltipProps={{
          onClose: () => {
            setError(undefined)
            setRepoMetadata(undefined)
          }
        }}
        disabled={disabled}
        renderMenu={
          <Menu>
            <Layout.Vertical
              className={css.formContainer}
              flex={{ justifyContent: 'center', alignItems: 'center' }}
              spacing="small"
              padding={'large'}>
              <Text font={{ variation: FontVariation.CARD_TITLE }}>{getString('cde.repository.pasteRepo')}</Text>
              <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_450}>
                {getString('cde.repository.pasterRepoSubtext')}
              </Text>
              <Container width={'100%'}>
                <TextInput
                  disabled={loading}
                  rightElementProps={{ size: 16, className: css.loadingIcon }}
                  rightElement={loading ? 'loading' : undefined}
                  className={css.urlInput}
                  placeholder="e.g https://github.com/microsoft/vscode-remote-try-python.git"
                  onChange={async event => {
                    const target = event.target as HTMLInputElement
                    await onChange(target.value)
                  }}
                />
                {error && <Text font={{ variation: FontVariation.FORM_MESSAGE_DANGER }}>{error}</Text>}
                {Boolean(repoMetadata) && (
                  <SelectRepositoryCard data={repoMetadata!} onChange={setFormikState} resetData={setRepoMetadata} />
                )}
              </Container>
              <Text font={{ variation: FontVariation.FORM_LABEL }}>{getString('cde.or')}</Text>
              <Text>
                {getString('cde.noRepo')} <Link to={'#'}> {getString('cde.createRepo')} </Link>
              </Text>
            </Layout.Vertical>
          </Menu>
        }
      />
    </Container>
  )
}
