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

import React, { useEffect, useMemo, useState } from 'react'
import {
  Button,
  Text,
  Container,
  Formik,
  Layout,
  Page,
  ButtonVariation,
  ButtonSize,
  FlexExpander,
  useToaster,
  Heading,
  TextInput,
  stringSubstitute
} from '@harnessio/uicore'
import cx from 'classnames'
import { noop } from 'lodash-es'
import { useMutate, useGet } from 'restful-react'
import { Intent, Color, FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { Dialog } from '@blueprintjs/core'
import { ProgressBar, Intent as IntentCore } from '@blueprintjs/core'
import { Icon } from '@harnessio/icons'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { JobProgress, useGetSpace } from 'services/code'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { REPO_EXPORT_STATE, getErrorMessage } from 'utils/Utils'
import { ACCESS_MODES, permissionProps, voidFn } from 'utils/Utils'
import type { ExportFormDataExtended } from 'utils/GitUtils'
import { useModalHook } from 'hooks/useModalHook'
import useSpaceSSE from 'hooks/useSpaceSSE'
import Harness from '../../../icons/Harness.svg?url'
import Upgrade from '../../../icons/Upgrade.svg?url'
import useDeleteSpaceModal from '../DeleteSpaceModal/DeleteSpaceModal'
import ExportForm from '../ExportForm/ExportForm'
import css from './GeneralSpaceSettings.module.scss'

export default function GeneralSpaceSettings() {
  const { space } = useGetRepositoryMetadata()
  const { openModal: openDeleteSpaceModal } = useDeleteSpaceModal()
  const { data, refetch } = useGetSpace({ space_ref: encodeURIComponent(space), lazy: !space })
  const [editName, setEditName] = useState(ACCESS_MODES.VIEW)
  const history = useHistory()
  const { routes, standalone, hooks } = useAppContext()
  const [upgrading, setUpgrading] = useState(false)
  const [editDesc, setEditDesc] = useState(ACCESS_MODES.VIEW)
  const [repoCount, setRepoCount] = useState(0)
  const [exportDone, setExportDone] = useState(false)
  const { showError, showSuccess } = useToaster()

  const { getString } = useStrings()
  const { mutate: patchSpace } = useMutate({
    verb: 'PATCH',
    path: `/api/v1/spaces/${space}`
  })
  const { mutate: updateName } = useMutate({
    verb: 'POST',
    path: `/api/v1/spaces/${space}/move`
  })
  const { data: exportProgressSpace, refetch: refetchExport } = useGet({
    path: `/api/v1/spaces/${space}/export-progress`
  })
  const countFinishedRepos = (): number => {
    return exportProgressSpace?.repos.filter((repo: JobProgress) => repo.state === REPO_EXPORT_STATE.FINISHED).length
  }

  const checkReposState = () => {
    return exportProgressSpace?.repos.every(
      (repo: JobProgress) =>
        repo.state === REPO_EXPORT_STATE.FINISHED ||
        repo.state === REPO_EXPORT_STATE.FAILED ||
        repo.state === REPO_EXPORT_STATE.CANCELED
    )
  }

  const checkExportIsRunning = () => {
    return exportProgressSpace?.repos.every(
      (repo: JobProgress) => repo.state === REPO_EXPORT_STATE.RUNNING || repo.state === REPO_EXPORT_STATE.SCHEDULED
    )
  }

  useEffect(() => {
    if (exportProgressSpace?.repos && checkExportIsRunning()) {
      setUpgrading(true)
      setRepoCount(exportProgressSpace?.repos.length)
      setExportDone(false)
    } else if (exportProgressSpace?.repos && checkReposState()) {
      setRepoCount(countFinishedRepos)
      setExportDone(true)
    }
  }, [exportProgressSpace]) // eslint-disable-line react-hooks/exhaustive-deps

  const events = useMemo(() => ['repository_export_completed'], [])

  useSpaceSSE({
    space,
    events,
    onEvent: () => {
      refetchExport()

      if (exportProgressSpace && checkReposState()) {
        setRepoCount(countFinishedRepos)
        setExportDone(true)
      } else if (exportProgressSpace?.repos && checkExportIsRunning()) {
        setUpgrading(true)
        setRepoCount(exportProgressSpace?.repos.length)
        setExportDone(false)
      }
    }
  })

  const ExportModal = () => {
    const [step, setStep] = useState(0)

    const { mutate: exportSpace } = useMutate({
      verb: 'POST',
      path: `/api/v1/spaces/${space}/export`
    })

    const handleExportSubmit = (formData: ExportFormDataExtended) => {
      try {
        setRepoCount(formData.repoCount)
        const exportPayload = {
          account_id: formData.accountId || '',
          org_identifier: formData.organization,
          project_identifier: formData.name,
          token: formData.token
        }
        exportSpace(exportPayload)
          .then(_ => {
            hideModal()
            setUpgrading(true)
            refetchExport()
          })
          .catch(_error => {
            showError(getErrorMessage(_error), 0, getString('failedToImportSpace'))
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, getString('failedToImportSpace'))
      }
    }

    return (
      <Dialog
        isOpen
        onClose={hideModal}
        enforceFocus={false}
        title={''}
        style={{
          width: 610,
          maxHeight: '95vh',
          overflow: 'auto'
        }}>
        <Layout.Vertical
          padding={{ left: 'xxxlarge' }}
          style={{ height: '100%' }}
          data-testid="add-target-to-flag-modal">
          <Heading level={3} font={{ variation: FontVariation.H3 }} margin={{ bottom: 'large' }}>
            <Layout.Horizontal className={css.upgradeHeader}>
              <img width={30} height={30} src={Harness} />
              <Text padding={{ left: 'small' }} font={{ variation: FontVariation.H4 }}>
                {step === 0 && <>{getString('exportSpace.upgradeHarness')}</>}
                {step === 1 && <>{getString('exportSpace.newProject')}</>}
                {step === 2 && <>{getString('exportSpace.upgradeConfirmation')}</>}
              </Text>
            </Layout.Horizontal>
          </Heading>
          <Container margin={{ right: 'xlarge' }}>
            <ExportForm
              hideModal={hideModal}
              step={step}
              setStep={setStep}
              handleSubmit={handleExportSubmit}
              loading={false}
              space={space}
            />
          </Container>
        </Layout.Vertical>
      </Dialog>
    )
  }
  const [openModal, hideModal] = useModalHook(ExportModal, [noop, space])
  const permEditResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY'
      },
      permissions: ['code_repo_edit']
    },
    [space]
  )
  const permDeleteResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY'
      },
      permissions: ['code_repo_delete']
    },
    [space]
  )
  return (
    <Container className={css.mainCtn}>
      <Page.Body>
        <Container padding="xlarge">
          <Formik
            formName="spaceGeneralSettings"
            initialValues={{
              name: data?.identifier,
              desc: data?.description
            }}
            onSubmit={voidFn(() => {
              // @typescript-eslint/no-empty-function
            })}>
            {formik => {
              return (
                <Layout.Vertical padding={{ top: 'medium' }}>
                  {upgrading ? (
                    <Container
                      height={exportDone ? 150 : 187}
                      color={Color.PRIMARY_BG}
                      padding="xlarge"
                      margin={{ bottom: 'medium' }}
                      className={css.generalContainer}>
                      <img width={148} height={148} src={Harness} className={css.harnessUpgradeWatermark} />
                      <Layout.Horizontal className={css.upgradeContainer}>
                        <img width={24} height={24} src={Harness} color={'blue'} />

                        <Text
                          padding={{ left: 'small' }}
                          font={{ variation: FontVariation.CARD_TITLE, size: 'medium' }}>
                          {exportDone
                            ? repoCount
                              ? getString('exportSpace.exportCompleted')
                              : getString('exportSpace.exportFailed')
                            : getString('exportSpace.upgradeProgress')}
                        </Text>
                      </Layout.Horizontal>
                      <Container padding={'xxlarge'}>
                        <Layout.Vertical spacing="large">
                          {exportDone ? null : <ProgressBar intent={IntentCore.PRIMARY} className={css.progressBar} />}
                          <Container padding={{ top: 'small' }}>
                            {exportDone ? (
                              <Text
                                icon={repoCount ? 'execution-success' : 'cross'}
                                iconProps={{
                                  size: 16,
                                  color: repoCount ? Color.GREEN_500 : Color.RED_500
                                }}>
                                <Text padding={{ left: 'small' }}>
                                  {repoCount
                                    ? (stringSubstitute(getString('exportSpace.exportRepoCompleted'), {
                                        repoCount
                                      }) as string)
                                    : getString('exportSpace.upgradeFailed')}
                                  {!repoCount && (
                                    <a target="_blank" rel="noreferrer" href="https://docs.gitness.com/support">
                                      <Icon className={css.icon} name="code-info" size={16} />
                                    </a>
                                  )}
                                </Text>
                              </Text>
                            ) : (
                              <Text
                                icon={'steps-spinner'}
                                iconProps={{
                                  size: 16,
                                  color: Color.GREY_300
                                }}>
                                <Text padding={{ left: 'small' }}>
                                  {
                                    stringSubstitute(getString('exportSpace.exportRepo'), {
                                      repoCount
                                    }) as string
                                  }
                                </Text>
                              </Text>
                            )}
                          </Container>
                        </Layout.Vertical>
                      </Container>
                    </Container>
                  ) : (
                    <Container
                      color={Color.PRIMARY_BG}
                      padding="xlarge"
                      margin={{ bottom: 'medium' }}
                      className={css.generalContainer}>
                      <img width={148} height={148} src={Harness} className={css.harnessWatermark} />
                      <Layout.Horizontal className={css.upgradeContainer}>
                        <img width={24} height={24} src={Harness} color={'blue'} />

                        <Text
                          padding={{ left: 'small' }}
                          font={{ variation: FontVariation.CARD_TITLE, size: 'medium' }}>
                          {getString('exportSpace.upgradeTitle')}
                        </Text>
                        <FlexExpander />
                        <Button
                          className={css.button}
                          variation={ButtonVariation.PRIMARY}
                          onClick={() => {
                            openModal()
                          }}
                          text={
                            <Layout.Horizontal
                              onClick={() => {
                                openModal()
                              }}>
                              <img width={16} height={16} src={Upgrade} />

                              <Text className={css.buttonText} color={Color.GREY_0}>
                                {getString('exportSpace.upgrade')}
                              </Text>
                            </Layout.Horizontal>
                          }
                          intent="success"
                          size={ButtonSize.MEDIUM}
                        />
                      </Layout.Horizontal>
                      <Text padding={{ top: 'large', left: 'xlarge' }} color={Color.GREY_500} font={{ size: 'small' }}>
                        {getString('exportSpace.upgradeContent')}
                        <a target="_blank" rel="noreferrer" href="https://developer.harness.io/docs/code-repository">
                          <Icon className={css.icon} name="code-info" size={16} />
                        </a>
                      </Text>
                    </Container>
                  )}
                  <Container padding="xlarge" margin={{ bottom: 'medium' }} className={css.generalContainer}>
                    <Layout.Horizontal padding={{ bottom: 'medium' }}>
                      <Container className={css.label}>
                        <Text padding={{ top: 'small' }} color={Color.GREY_600} className={css.textSize}>
                          {getString('name')}
                        </Text>
                      </Container>
                      <Container className={css.content}>
                        {editName === ACCESS_MODES.EDIT ? (
                          <Layout.Horizontal>
                            <TextInput
                              name="name"
                              value={formik.values.name || data?.identifier}
                              className={cx(css.textContainer, css.textSize)}
                              onChange={evt => {
                                formik.setFieldValue('name', (evt.currentTarget as HTMLInputElement)?.value)
                              }}
                            />
                            <Layout.Horizontal className={css.buttonContainer}>
                              <Button
                                className={css.saveBtn}
                                margin={{ right: 'medium' }}
                                type="submit"
                                text={getString('save')}
                                variation={ButtonVariation.SECONDARY}
                                size={ButtonSize.SMALL}
                                onClick={() => {
                                  updateName({ uid: formik.values?.name })
                                    .then(() => {
                                      showSuccess(getString('spaceUpdate'))
                                      history.push(routes.toCODESpaceSettings({ space: formik.values?.name as string }))
                                      setEditName(ACCESS_MODES.VIEW)
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
                                  formik.setFieldValue('name', data?.identifier)
                                  setEditName(ACCESS_MODES.VIEW)
                                }}
                              />
                            </Layout.Horizontal>
                          </Layout.Horizontal>
                        ) : (
                          <Text color={Color.GREY_800} className={css.textSize}>
                            {formik?.values?.name || data?.identifier}
                            <Button
                              className={css.textSize}
                              text={getString('edit')}
                              icon="Edit"
                              variation={ButtonVariation.LINK}
                              onClick={() => {
                                setEditName(ACCESS_MODES.EDIT)
                              }}
                              {...permissionProps(permEditResult, standalone)}
                            />
                          </Text>
                        )}
                      </Container>
                    </Layout.Horizontal>
                    <Layout.Horizontal>
                      <Container className={css.label}>
                        <Text padding={{ top: 'small' }} color={Color.GREY_600} className={css.textSize}>
                          {getString('description')}
                        </Text>
                      </Container>
                      <Container className={css.content}>
                        {editDesc === ACCESS_MODES.EDIT ? (
                          <Layout.Horizontal>
                            <TextInput
                              onChange={evt => {
                                formik.setFieldValue('desc', (evt.currentTarget as HTMLInputElement)?.value)
                              }}
                              value={formik.values.desc || data?.description}
                              name="desc"
                              className={cx(css.textContainer, css.textSize)}
                            />
                            <Layout.Horizontal className={css.buttonContainer}>
                              <Button
                                className={cx(css.saveBtn, css.textSize)}
                                margin={{ right: 'medium' }}
                                type="submit"
                                text={getString('save')}
                                variation={ButtonVariation.SECONDARY}
                                size={ButtonSize.SMALL}
                                onClick={() => {
                                  patchSpace({ description: formik.values?.desc })
                                    .then(() => {
                                      showSuccess(getString('spaceUpdate'))
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
                                  formik.setFieldValue('desc', data?.description)
                                  setEditDesc(ACCESS_MODES.VIEW)
                                }}
                              />
                            </Layout.Horizontal>
                          </Layout.Horizontal>
                        ) : (
                          <Text className={css.textSize} color={Color.GREY_800}>
                            {formik?.values?.desc || data?.description}
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
                  </Container>
                  <Container padding="large" className={css.generalContainer}>
                    <Container className={css.deleteContainer}>
                      <Layout.Vertical className={css.verticalContainer}>
                        <Text icon="main-trash" color={Color.GREY_600} font={{ size: 'small' }}>
                          {getString('dangerDeleteRepo')}
                        </Text>
                        <Layout.Horizontal
                          padding={{ top: 'medium', left: 'medium' }}
                          flex={{ justifyContent: 'space-between' }}>
                          <Container className={css.yellowContainer}>
                            <Text
                              icon="main-issue"
                              iconProps={{ size: 16, color: Color.ORANGE_700, margin: { right: 'small' } }}
                              padding={{ left: 'large', right: 'large', top: 'small', bottom: 'small' }}
                              color={Color.WARNING}>
                              {getString('spaceSetting.intentText', {
                                space: data?.identifier
                              })}
                            </Text>
                          </Container>
                          <Button
                            className={css.deleteBtn}
                            margin={{ right: 'medium' }}
                            disabled={false}
                            intent={Intent.DANGER}
                            onClick={() => {
                              openDeleteSpaceModal()
                            }}
                            variation={ButtonVariation.SECONDARY}
                            text={getString('deleteSpace')}
                            {...permissionProps(permDeleteResult, standalone)}></Button>
                        </Layout.Horizontal>
                      </Layout.Vertical>
                    </Container>
                  </Container>
                </Layout.Vertical>
              )
            }}
          </Formik>
        </Container>
      </Page.Body>
    </Container>
  )
}
