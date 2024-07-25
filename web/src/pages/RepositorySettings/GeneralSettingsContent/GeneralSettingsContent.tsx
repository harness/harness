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
import {
  Container,
  Layout,
  Text,
  Button,
  ButtonVariation,
  Formik,
  useToaster,
  ButtonSize,
  FormInput,
  Dialog,
  StringSubstitute
} from '@harnessio/uicore'
import cx from 'classnames'
import { Color, FontVariation, Intent } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import { noop } from 'lodash-es'
import { useMutate, useGet } from 'restful-react'
import { Render } from 'react-jsx-match'
import { ACCESS_MODES, getErrorMessage, permissionProps, voidFn } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import type { RepoRepositoryOutput } from 'services/code'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { RepoVisibility } from 'utils/GitUtils'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import { useModalHook } from 'hooks/useModalHook'
import useDeleteRepoModal from './DeleteRepoModal/DeleteRepoModal'
import useDefaultBranchModal from './DefaultBranchModal/DefaultBranchModal'
import Private from '../../../icons/private.svg?url'
import css from '../RepositorySettings.module.scss'

interface GeneralSettingsProps {
  repoMetadata: RepoRepositoryOutput | undefined
  refetch: () => void
  gitRef: string
  isRepositoryEmpty: boolean
}

const GeneralSettingsContent = (props: GeneralSettingsProps) => {
  const { repoMetadata, refetch, gitRef, isRepositoryEmpty } = props
  const { openModal: openDeleteRepoModal } = useDeleteRepoModal()
  const [currentGitRef, setCurrentGitRef] = useState(gitRef)
  const [editDesc, setEditDesc] = useState(ACCESS_MODES.VIEW)
  const [defaultBranch, setDefaultBranch] = useState(ACCESS_MODES.VIEW)
  const { openModal: openDefaultBranchModal } = useDefaultBranchModal({ currentGitRef, setDefaultBranch, refetch })
  const { showError, showSuccess } = useToaster()

  const space = useGetSpaceParam()
  const { standalone, hooks, isPublicAccessEnabledOnResources } = useAppContext()
  const { getString } = useStrings()
  const currRepoVisibility = repoMetadata?.is_public === true ? RepoVisibility.PUBLIC : RepoVisibility.PRIVATE

  const [repoVis, setRepoVis] = useState<RepoVisibility>(currRepoVisibility)
  const [enablePublicRepo, setEnablePublicRepo] = useState(false)
  const { mutate } = useMutate({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoMetadata?.path}/+/`
  })

  const { mutate: changeVisibility } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata?.path}/+/public-access`
  })

  const permEditResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: ['code_repo_edit']
    },
    [space]
  )
  const permDeleteResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: ['code_repo_delete']
    },
    [space]
  )
  const { data: systemConfig } = useGet({ path: 'api/v1/system/config' })

  useEffect(() => {
    if (systemConfig) {
      setEnablePublicRepo(systemConfig.public_resource_creation_enabled)
    }
  }, [systemConfig])

  const ModalComponent: React.FC = () => {
    return (
      <Dialog
        className={css.dialogContainer}
        style={{ width: 585, maxHeight: '95vh', overflow: 'auto' }}
        title={<Text font={{ variation: FontVariation.H4 }}>{getString('changeRepoVis')}</Text>}
        isOpen
        onClose={hideModal}>
        <Layout.Vertical spacing="xlarge">
          <Text>
            <StringSubstitute
              str={getString('changeRepoVisContent')}
              vars={{
                repoVis: <span className={css.text}>{repoVis}</span>
              }}
            />
          </Text>
          <Container
            intent="warning"
            background="yellow100"
            border={{
              color: 'orange500'
            }}
            margin={{ top: 'medium', bottom: 'medium' }}>
            <Text
              icon="warning-outline"
              iconProps={{ size: 16, margin: { right: 'small' } }}
              padding={{ left: 'large', right: 'large', top: 'small', bottom: 'small' }}
              color={Color.WARNING}>
              {repoVis === RepoVisibility.PUBLIC
                ? getString('createRepoModal.publicWarning')
                : getString('createRepoModal.privateLabel')}
            </Text>
          </Container>
          <Layout.Horizontal className={css.buttonContainer}>
            <Button
              margin={{ right: 'medium' }}
              type="submit"
              text={
                <StringSubstitute
                  str={getString('confirmRepoVisButton')}
                  vars={{
                    repoVis: <span className={css.text}>{repoVis}</span>
                  }}
                />
              }
              variation={ButtonVariation.PRIMARY}
              onClick={() => {
                changeVisibility({ is_public: repoVis === RepoVisibility.PUBLIC ? true : false })
                  .then(() => {
                    showSuccess(getString('repoUpdate'))
                    hideModal()
                    refetch()
                  })
                  .catch(err => {
                    showError(getErrorMessage(err))
                  })
                refetch()
              }}
            />
            <Button
              text={getString('cancel')}
              variation={ButtonVariation.TERTIARY}
              onClick={() => {
                hideModal()
              }}
            />
          </Layout.Horizontal>
        </Layout.Vertical>
      </Dialog>
    )
  }
  const [openModal, hideModal] = useModalHook(ModalComponent, [voidFn(noop)])

  return (
    <Formik
      formName="repoGeneralSettings"
      initialValues={{
        name: repoMetadata?.identifier,
        desc: repoMetadata?.description,
        defaultBranch: repoMetadata?.default_branch,
        isPublic: currRepoVisibility
      }}
      onSubmit={voidFn(mutate)}>
      {formik => {
        return (
          <Layout.Vertical padding={{ top: 'medium' }}>
            <Container padding="large" margin={{ bottom: 'medium' }} className={css.generalContainer}>
              <Layout.Horizontal padding={{ bottom: 'medium' }}>
                <Container className={css.label}>
                  <Text color={Color.GREY_600} className={css.textSize}>
                    {getString('name')}
                  </Text>
                </Container>
                <Container className={css.content}>
                  <Text color={Color.GREY_800} className={css.textSize}>
                    {repoMetadata?.identifier}
                  </Text>
                </Container>
              </Layout.Horizontal>
              <Layout.Horizontal padding={{ bottom: 'medium' }}>
                <Container className={cx(css.label, css.descText)}>
                  <Text color={Color.GREY_600} className={css.textSize}>
                    {getString('description')}
                  </Text>
                </Container>
                <Container className={css.content}>
                  {editDesc === ACCESS_MODES.EDIT ? (
                    <Layout.Vertical className={css.editContainer} margin={{ top: 'xlarge', bottom: 'xlarge' }}>
                      <FormInput.TextArea
                        className={cx(css.textContainer, css.textSize)}
                        placeholder={getString('enterRepoDescription')}
                        name="desc"
                      />
                      <Layout.Horizontal className={css.buttonContainer}>
                        <Button
                          type="submit"
                          text={getString('save')}
                          variation={ButtonVariation.SECONDARY}
                          size={ButtonSize.SMALL}
                          onClick={() => {
                            mutate({ description: formik.values?.desc?.replace(/\n/g, ' ') })
                              .then(() => {
                                showSuccess(getString('repoUpdate'))
                                setEditDesc(ACCESS_MODES.VIEW)
                                refetch()
                              })
                              .catch(err => {
                                showError(getErrorMessage(err))
                              })
                          }}
                        />
                        <Button
                          text={getString('cancel')}
                          variation={ButtonVariation.TERTIARY}
                          size={ButtonSize.SMALL}
                          onClick={() => {
                            formik.setFieldValue('desc', repoMetadata?.description)
                            setEditDesc(ACCESS_MODES.VIEW)
                          }}
                        />
                      </Layout.Horizontal>
                    </Layout.Vertical>
                  ) : (
                    <Text color={Color.GREY_800} className={cx(css.textSize, css.description)}>
                      {formik?.values?.desc || repoMetadata?.description}
                      <Button
                        className={css.textSize}
                        text={getString('edit')}
                        icon="Edit"
                        variation={ButtonVariation.LINK}
                        onClick={() => {
                          setEditDesc(ACCESS_MODES.EDIT)
                        }}
                        {...permissionProps(permEditResult, standalone)}
                      />
                    </Text>
                  )}
                </Container>
              </Layout.Horizontal>
              <Layout.Horizontal>
                <Container className={cx(css.label, css.descText)}>
                  <Text color={Color.GREY_600} className={css.textSize}>
                    {getString('defaultBranchTitle')}
                  </Text>
                </Container>
                <Container className={css.content}>
                  <Layout.Horizontal className={css.editContainer}>
                    {repoMetadata && (
                      <BranchTagSelect
                        forBranchesOnly={true}
                        disableBranchCreation={true}
                        disableViewAllBranches={isRepositoryEmpty}
                        disabled={defaultBranch !== ACCESS_MODES.EDIT}
                        hidePopoverContent={defaultBranch !== ACCESS_MODES.EDIT}
                        repoMetadata={repoMetadata}
                        margin={{ right: 'large' }}
                        gitRef={currentGitRef}
                        size={ButtonSize.SMALL}
                        onSelect={ref => {
                          setCurrentGitRef(ref)
                        }}
                      />
                    )}
                    {defaultBranch === ACCESS_MODES.EDIT ? (
                      <>
                        <Button
                          margin={{ right: 'small' }}
                          text={getString('save')}
                          disabled={currentGitRef === repoMetadata?.default_branch}
                          variation={ButtonVariation.PRIMARY}
                          size={ButtonSize.SMALL}
                          onClick={() => {
                            openDefaultBranchModal()
                          }}
                        />
                        <Button
                          text={getString('cancel')}
                          variation={ButtonVariation.TERTIARY}
                          size={ButtonSize.SMALL}
                          onClick={() => {
                            setCurrentGitRef(repoMetadata?.default_branch as string)
                            setDefaultBranch(ACCESS_MODES.VIEW)
                          }}
                        />
                      </>
                    ) : (
                      <>
                        <Button
                          className={css.saveBtn}
                          margin={{ right: 'medium' }}
                          text={getString('switchBranch')}
                          variation={ButtonVariation.SECONDARY}
                          size={ButtonSize.SMALL}
                          onClick={() => {
                            setDefaultBranch(ACCESS_MODES.EDIT)
                          }}
                          {...permissionProps(permEditResult, standalone)}
                        />
                      </>
                    )}
                  </Layout.Horizontal>
                </Container>
              </Layout.Horizontal>
            </Container>
            <Render when={enablePublicRepo && isPublicAccessEnabledOnResources}>
              <Container padding="large" margin={{ bottom: 'medium' }} className={css.generalContainer}>
                <Layout.Horizontal padding={{ bottom: 'medium' }}>
                  <Container className={css.label}>
                    <Text color={Color.GREY_600} font={{ size: 'small' }}>
                      {getString('visibility')}
                    </Text>
                  </Container>
                  <Container className={css.content}>
                    <FormInput.RadioGroup
                      name="isPublic"
                      label=""
                      onChange={evt => {
                        setRepoVis((evt.target as HTMLInputElement).value as RepoVisibility)
                      }}
                      {...permissionProps(permEditResult, standalone)}
                      className={css.radioContainer}
                      items={[
                        {
                          label: (
                            <Container>
                              <Layout.Horizontal>
                                <Icon
                                  className={css.iconContainer}
                                  name="git-clone-step"
                                  size={20}
                                  margin={{ left: 'small', right: 'medium' }}
                                />
                                <Container>
                                  <Layout.Vertical spacing="xsmall">
                                    <Text font={{ size: 'small' }}>{getString('public')}</Text>
                                    <Text font={{ variation: FontVariation.TINY }}>
                                      {getString('createRepoModal.publicLabel')}
                                    </Text>
                                  </Layout.Vertical>
                                </Container>
                              </Layout.Horizontal>
                            </Container>
                          ),

                          value: RepoVisibility.PUBLIC
                        },
                        {
                          label: (
                            <Container>
                              <Layout.Horizontal>
                                <Container className={css.iconContainer} margin={{ left: 'small', right: 'medium' }}>
                                  <img width={20} height={20} src={Private} />
                                </Container>
                                <Container margin={{ left: 'small' }}>
                                  <Layout.Vertical spacing="xsmall">
                                    <Text font={{ size: 'small' }}>{getString('private')}</Text>
                                    <Text font={{ variation: FontVariation.TINY }}>
                                      {getString('createRepoModal.privateLabel')}
                                    </Text>
                                  </Layout.Vertical>
                                </Container>
                              </Layout.Horizontal>
                            </Container>
                          ),
                          value: RepoVisibility.PRIVATE
                        }
                      ]}
                    />
                    <hr className={css.dividerContainer} />
                    <Layout.Horizontal className={css.buttonContainer}>
                      {repoVis !== currRepoVisibility ? (
                        <Button
                          margin={{ right: 'medium' }}
                          type="submit"
                          text={getString('save')}
                          variation={ButtonVariation.PRIMARY}
                          size={ButtonSize.SMALL}
                          onClick={() => {
                            setRepoVis(formik.values.isPublic)
                            openModal()
                          }}
                          {...permissionProps(permEditResult, standalone)}
                        />
                      ) : null}
                    </Layout.Horizontal>
                  </Container>
                </Layout.Horizontal>
              </Container>
            </Render>
            <Container padding="medium" className={css.generalContainer}>
              <Container className={css.deleteContainer}>
                <Text icon="main-trash" color={Color.GREY_600} font={{ size: 'small' }}>
                  {getString('dangerDeleteRepo')}
                </Text>
                <Button
                  intent={Intent.DANGER}
                  onClick={() => {
                    openDeleteRepoModal()
                  }}
                  variation={ButtonVariation.SECONDARY}
                  text={getString('delete')}
                  {...permissionProps(permDeleteResult, standalone)}></Button>
              </Container>
            </Container>
          </Layout.Vertical>
        )
      }}
    </Formik>
  )
}

export default GeneralSettingsContent
