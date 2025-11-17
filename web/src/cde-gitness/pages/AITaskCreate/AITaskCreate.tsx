/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import React, { useCallback, useState } from 'react'
import {
  Breadcrumbs,
  Card,
  Container,
  Layout,
  Page,
  Text,
  FormInput,
  Button,
  ButtonVariation,
  Formik,
  FormikForm,
  useToaster
} from '@harnessio/uicore'
import { Menu, MenuItem, PopoverInteractionKind, PopoverPosition, Popover, TextArea } from '@blueprintjs/core'
import * as yup from 'yup'
import { useHistory } from 'react-router-dom'
import { Color, FontVariation } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import AIbird from 'cde-gitness/assests/AIbird.svg?url'
import SelectContextDialog from 'cde-gitness/components/SelectContextDialog/SelectContextDialog'
import type { TypesGitspaceConfig } from 'services/cde'
import { useCreateAITask, type EnumAIAgent } from 'services/cde'
import { getIconByRepoType, getRepoNameFromURL } from 'cde-gitness/utils/SelectRepository.utils'
import { AIAgentEnum } from 'cde-gitness/constants/index'
import codeSandboxLogo from 'cde-gitness/assests/codeSandboxLogo.svg?url'
import claudeIcon from 'cde-gitness/assests/claude.svg?url'
import { getErrorMessage } from 'utils/Utils'
import css from './AITaskCreate.module.scss'

const AITaskCreate = () => {
  const space = useGetSpaceParam()
  const { getString } = useStrings()
  const { standalone, accountInfo, routes, currentUser } = useAppContext()
  const { orgIdentifier = '', projectIdentifier = '', accountIdentifier = '' } = useGetCDEAPIParams()
  const userName = currentUser?.display_name || 'there'
  const history = useHistory()
  const { showSuccess, showError } = useToaster()
  const [isContextOpen, setIsContextOpen] = useState(false)
  const openContext = useCallback(() => setIsContextOpen(true), [])
  const closeContext = useCallback(() => setIsContextOpen(false), [])
  const { mutate: createAITask, loading: creatingTask } = useCreateAITask({
    accountIdentifier: accountIdentifier,
    orgIdentifier: orgIdentifier,
    projectIdentifier: projectIdentifier
  })

  type AITaskFormValues = {
    initial_prompt: string
    gitspace_config_id: string
    ai_agent?: EnumAIAgent
    gitspace?: TypesGitspaceConfig | undefined
  }

  const formInitialValues: AITaskFormValues = {
    initial_prompt: '',
    gitspace_config_id: '',
    ai_agent: undefined,
    gitspace: undefined
  }

  const validationSchema = yup.object().shape({
    initial_prompt: yup.string().trim().required(),
    gitspace_config_id: yup.string().trim().required(),
    ai_agent: yup.string().oneOf([AIAgentEnum.CLAUDE_CODE]).required()
  })

  const handleSubmit = async (values: AITaskFormValues) => {
    const identifier = 'aitask'
    const payload = {
      name: '',
      identifier,
      initial_prompt: values.initial_prompt,
      gitspace_config_id: values.gitspace_config_id,
      ai_agent: values.ai_agent || (AIAgentEnum.CLAUDE_CODE as EnumAIAgent),
      space_ref: space
    }
    try {
      const response = await createAITask(payload)
      showSuccess(getString('cde.aiTasks.create.aiTaskCreateSuccess'))
      history.push(
        `${routes.toCDEAITaskDetail({
          space,
          aitaskId: response.identifier || ''
        })}?redirectFrom=login`
      )
    } catch (error) {
      showError(getString('cde.aiTasks.create.aiTaskCreateFailed'))
      showError(getErrorMessage(error))
    }
  }
  const getBreadcrumbLinks = () => {
    if (standalone) {
      return [
        { url: routes.toCDEAITasks({ space }), label: getString('cde.aiTasks.tasks') },
        { url: routes.toCDEAITaskCreate({ space }), label: getString('cde.aiTasks.create.createTask') }
      ]
    }

    return [
      {
        url: `/account/${accountIdentifier}/module/cde`,
        label: `Account: ${accountInfo?.name || accountIdentifier}`
      },
      {
        url: `/account/${accountIdentifier}/module/cde/orgs/${orgIdentifier}`,
        label: `Organization: ${orgIdentifier}`
      },
      {
        url: `/account/${accountIdentifier}/module/cde/orgs/${orgIdentifier}/projects/${projectIdentifier}`,
        label: `Project: ${projectIdentifier}`
      },
      {
        url: routes.toCDEAITasks({ space }),
        label: getString('cde.aiTasks.tasks')
      },
      {
        url: routes.toCDEAITaskCreate({ space }),
        label: getString('cde.aiTasks.create.createTask')
      }
    ]
  }

  const renderSelectedContext = useCallback(
    (gs?: TypesGitspaceConfig) => {
      if (!gs) return null
      const repoName = getRepoNameFromURL(gs.code_repo_url || '') || ''
      const branch = gs.branch || ''

      return (
        <Container className={css.contextRow} onClick={openContext}>
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }} className={css.left}>
            <img src={codeSandboxLogo} alt="CodeSandbox" height={26} width={26} />
            <Layout.Vertical spacing="xsmall">
              <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
                {getString('cde.aiTasks.create.selectedGitspace')}
              </Text>
              <Text color={Color.BLACK} font={{ variation: FontVariation.BODY2_SEMI }} lineClamp={1}>
                {gs.identifier || gs.name}
              </Text>
            </Layout.Vertical>
          </Layout.Horizontal>

          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'center' }}>
            <Layout.Horizontal spacing="xsmall" flex={{ alignItems: 'center', justifyContent: 'center' }}>
              <Container height={14} width={14}>
                {getIconByRepoType({ repoType: gs.code_repo_type, height: 16 })}
              </Container>
              <Text color={Color.GREY_800} lineClamp={1} font={{ variation: FontVariation.SMALL }}>
                {repoName}
              </Text>
              <Text>:</Text>
              <Text
                icon="git-branch"
                iconProps={{ size: 12 }}
                color={Color.GREY_800}
                lineClamp={1}
                font={{ variation: FontVariation.SMALL }}>
                {branch}
              </Text>
            </Layout.Horizontal>
            <Icon name="chevron-down" size={14} className={css.rightIcon} />
          </Layout.Horizontal>
        </Container>
      )
    },
    [getString, openContext]
  )

  const renderAddContext = useCallback(
    () => (
      <Container className={css.contextRow} onClick={openContext}>
        <Container className={css.left}>
          <Icon name="cde_snapshot" size={32} />
          <Text className={css.title} color={Color.BLACK} font={{ variation: FontVariation.BODY2_SEMI }}>
            {getString('cde.aiTasks.create.addContext')}
          </Text>
          <Text
            className={css.subtitle}
            color={Color.GREY_500}
            font={{ variation: FontVariation.BODY2, weight: 'light' }}>
            {`(${getString('cde.aiTasks.create.selectActiveGitspace')})`}
          </Text>
        </Container>
        <Icon name="chevron-down" size={14} className={css.rightIcon} />
      </Container>
    ),
    [getString, openContext]
  )

  return (
    <>
      <Page.Header
        title={getString('cde.aiTasks.create.createTask')}
        breadcrumbs={<Breadcrumbs links={getBreadcrumbLinks()} />}
      />
      <Formik
        formName={getString('cde.aiTasks.create.createTask')}
        initialValues={formInitialValues}
        enableReinitialize={true}
        validationSchema={validationSchema}
        validateOnChange={true}
        validateOnBlur={false}
        onSubmit={handleSubmit}>
        {formik => {
          const errors = formik.errors || {}
          const missing: string[] = []
          if (errors.gitspace_config_id) missing.push('Context')
          if (errors.ai_agent) missing.push('AI agent')
          if (errors.initial_prompt) missing.push('Initial prompt')

          let errorMessage = ''
          if (missing.length === 1) {
            errorMessage = `${missing[0]} is required`
          } else if (missing.length === 2) {
            errorMessage = `${missing[0]} and ${missing[1]} are required`
          } else if (missing.length === 3) {
            errorMessage = `${missing[0]}, ${missing[1]} and ${missing[2]} are required`
          }

          return (
            <>
              <Page.Body className={css.pageContainer}>
                <Container className={css.contentContainer}>
                  <Layout.Vertical spacing={'small'} flex={{ justifyContent: 'center' }} className={css.titleContainer}>
                    <img src={AIbird} alt="AIbird" height={80} width={80} />
                    <Text font={{ variation: FontVariation.H3 }} color={Color.BLACK}>
                      {getString('cde.aiTasks.create.welcome', { userName })}
                    </Text>
                    <Text font={{ variation: FontVariation.BODY2, weight: 'light' }} color={Color.GREY_500}>
                      {getString('cde.aiTasks.create.createTaskNote')}
                    </Text>
                  </Layout.Vertical>
                  <FormikForm className={css.fomikForm}>
                    <Card className={css.cardContainer}>
                      <Container className={css.cardHeader}>
                        <Container className={css.contextContainer}>
                          {formik.values.gitspace ? renderSelectedContext(formik.values.gitspace) : renderAddContext()}
                        </Container>
                      </Container>
                      <Container className={css.cardBody}>
                        <TextArea
                          className={css.promptInput}
                          placeholder={getString('cde.aiTasks.create.promptPlaceholder')}
                          fill
                          growVertically={false}
                          value={formik.values.initial_prompt}
                          onChange={e => formik.setFieldValue('initial_prompt', e.currentTarget.value)}
                        />
                      </Container>
                      <Container className={css.cardFooter}>
                        <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }} width={'100%'}>
                          <Container className={css.footerErrors} role="alert" aria-live="polite">
                            {formik.submitCount > 0 && errorMessage ? (
                              <Text
                                icon="circle-cross"
                                iconProps={{ color: Color.RED_700, size: 12 }}
                                color={Color.RED_500}
                                font={{ variation: FontVariation.BODY }}>
                                {errorMessage}
                              </Text>
                            ) : null}
                          </Container>
                          <Layout.Horizontal
                            spacing="small"
                            className={css.footerRight}
                            flex={{ alignItems: 'center', justifyContent: 'flex-end' }}>
                            <Container>
                              <FormInput.CustomRender
                                name=""
                                className={css.formInput}
                                render={() => (
                                  <Popover
                                    interactionKind={PopoverInteractionKind.CLICK}
                                    position={PopoverPosition.BOTTOM}
                                    popoverClassName={css.aiMenuContainer}
                                    content={
                                      <Container width={200} padding="xsmall">
                                        <Menu>
                                          <MenuItem
                                            active={formik.values.ai_agent === AIAgentEnum.CLAUDE_CODE}
                                            text={
                                              <Text font={{ size: 'normal', weight: 'bold' }}>
                                                {getString('cde.aiTasks.create.claudeAI')}
                                              </Text>
                                            }
                                            icon={
                                              <div className={css.selectAgentIcon}>
                                                <img src={claudeIcon} height={16} width={16} />
                                              </div>
                                            }
                                            onClick={e => {
                                              e.preventDefault()
                                              e.stopPropagation()
                                              formik.setFieldValue('ai_agent', AIAgentEnum.CLAUDE_CODE)
                                            }}
                                          />
                                        </Menu>
                                      </Container>
                                    }>
                                    <Container className={css.aiSelect} role="button" tabIndex={0}>
                                      <Layout.Horizontal
                                        spacing="xsmall"
                                        flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                                        {formik.values.ai_agent && <img src={claudeIcon} height={16} width={16} />}
                                        <Text font={'normal'}>
                                          {formik.values.ai_agent
                                            ? getString('cde.aiTasks.create.claudeAI')
                                            : getString('cde.aiTasks.create.selectAIAgent')}
                                        </Text>
                                        <Icon name="chevron-down" size={14} className={css.aiChevron} />
                                      </Layout.Horizontal>
                                    </Container>
                                  </Popover>
                                )}
                              />
                            </Container>
                            <Button
                              variation={ButtonVariation.AI}
                              text={getString('cde.aiTasks.create.runTask')}
                              rightIcon="chevron-right"
                              iconProps={{ className: css.rightIcon }}
                              type="submit"
                              disabled={creatingTask}
                            />
                          </Layout.Horizontal>
                        </Layout.Horizontal>
                      </Container>
                    </Card>
                  </FormikForm>
                </Container>
              </Page.Body>
              <SelectContextDialog
                isOpen={isContextOpen}
                onClose={closeContext}
                onApply={gs => {
                  formik.setFieldValue('gitspace', gs || undefined)
                  formik.setFieldValue('gitspace_config_id', gs?.identifier || '')
                  setIsContextOpen(false)
                }}
                selectedGitspaceId={formik.values.gitspace?.identifier}
                title={getString('cde.aiTasks.create.selectContext')}
              />
            </>
          )
        }}
      </Formik>
    </>
  )
}

export default AITaskCreate
