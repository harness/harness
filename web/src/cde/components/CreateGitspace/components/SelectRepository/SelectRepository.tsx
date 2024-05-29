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

import React, { useState } from 'react'
import { Button, ButtonVariation, Card, FormInput, Formik, FormikForm, Layout, Text } from '@harnessio/uicore'
import { Menu, MenuItem } from '@blueprintjs/core'
import { Color, FontVariation } from '@harnessio/design-system'
import { Link } from 'react-router-dom'
import { noop } from 'lodash-es'
import { Icon } from '@harnessio/icons'
import { useFormikContext } from 'formik'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { OpenapiGetCodeRepositoryResponse, useGetCodeRepository } from 'services/cde'
import { GitspaceSelect } from 'cde/components/GitspaceSelect/GitspaceSelect'
import { useStrings } from 'framework/strings'
import { CodeRepoAccessType } from 'cde/constants'
import { getErrorMessage } from 'utils/Utils'
import type { GitspaceFormInterface } from '../../CreateGitspace'
import css from './SelectRepository.module.scss'

export const getRepoNameFromURL = (repoURL?: string) => {
  const repoURLSplit = repoURL?.split('/')
  return repoURLSplit?.[repoURLSplit?.length - 1]
}

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

export const SelectRepository = () => {
  const { getString } = useStrings()
  const { values, errors, setFieldValue: onChange } = useFormikContext<GitspaceFormInterface>()

  const { codeRepoUrl } = values
  const space = useGetSpaceParam()

  const [validatedOnce, setValidatedOnce] = useState(false)
  const [repoMetadata, setRepoMetadata] = useState<OpenapiGetCodeRepositoryResponse | undefined>()

  const { mutate, loading } = useGetCodeRepository({
    accountIdentifier: space?.split('/')[0],
    orgIdentifier: space?.split('/')[1],
    projectIdentifier: space?.split('/')[2]
  })

  return (
    <GitspaceSelect
      text={<RepositoryText repoURL={codeRepoUrl} />}
      icon={'code'}
      errorMessage={errors.codeRepoId}
      formikName="codeRepoId"
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
            <Formik
              formLoading={loading}
              onSubmit={() => noop()}
              formName={'publicURL'}
              initialValues={{ url: codeRepoUrl }}
              validate={async ({ url }) => {
                if (!url) {
                  return {}
                }
                let errorMessages = undefined
                try {
                  const response = await mutate({ url })
                  if (response?.access_type === CodeRepoAccessType.PRIVATE) {
                    errorMessages = { url: getString('cde.repository.privateRepoWarning') }
                  }
                  setRepoMetadata(response)
                } catch (error) {
                  errorMessages = { url: getErrorMessage(error) }
                }
                setValidatedOnce(true)
                return errorMessages
              }}>
              {formikProps => {
                if (!formikProps.touched.url && validatedOnce) {
                  formikProps.setFieldTouched('url', true)
                }
                return (
                  <FormikForm>
                    <FormInput.Text
                      name="url"
                      className={css.urlInput}
                      placeholder="e.g https://github.com/orkohunter/idp"
                    />
                    {!!repoMetadata && (
                      <Card className={css.repometadataCard}>
                        {repoMetadata?.access_type === CodeRepoAccessType.PRIVATE ? (
                          <Layout.Vertical
                            className={css.privateCard}
                            flex={{ justifyContent: 'center', alignItems: 'center' }}
                            spacing="medium">
                            <Icon name="lock" size={20} />
                            <Text>{getString('cde.repository.privateRepoWarning')}</Text>
                            <Button variation={ButtonVariation.PRIMARY}>{`${getString('cde.repository.continueWith')} ${
                              repoMetadata?.repo_type
                            }`}</Button>
                          </Layout.Vertical>
                        ) : (
                          <Menu className={css.repometadata}>
                            <MenuItem
                              className={css.metadataItem}
                              onClick={() => {
                                onChange('codeRepoUrl', repoMetadata?.url || '')
                                onChange('codeRepoType', repoMetadata?.repo_type || '')
                                onChange('codeRepoId', repoMetadata?.url || '')
                              }}
                              text={
                                <Layout.Horizontal
                                  spacing="large"
                                  flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                                  <Text font={'small'} lineClamp={1}>
                                    {getRepoNameFromURL(repoMetadata?.url)}
                                  </Text>
                                  <Text className={css.publicTag} font={'xsmall'}>
                                    {repoMetadata?.access_type}
                                  </Text>
                                </Layout.Horizontal>
                              }
                            />
                          </Menu>
                        )}
                      </Card>
                    )}
                  </FormikForm>
                )
              }}
            </Formik>
            <Text font={{ variation: FontVariation.FORM_LABEL }}>{getString('cde.or')}</Text>
            <Text>
              {getString('cde.noRepo')} <Link to={'#'}> {getString('cde.createRepo')} </Link>
            </Text>
          </Layout.Vertical>
        </Menu>
      }
    />
  )
}
