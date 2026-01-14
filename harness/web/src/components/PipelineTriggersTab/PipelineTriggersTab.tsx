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

import {
  ButtonVariation,
  FlexExpander,
  Layout,
  Text,
  Formik,
  FormikForm,
  Container,
  Button,
  useToaster,
  Checkbox,
  FormInput
} from '@harnessio/uicore'
import React from 'react'
import { useGet, useMutate } from 'restful-react'
import cx from 'classnames'
import { Color, Intent } from '@harnessio/design-system'
import * as yup from 'yup'
import { Icon } from '@harnessio/icons'
import { String, useStrings } from 'framework/strings'
import type { EnumTriggerAction, OpenapiUpdateTriggerRequest, TypesTrigger } from 'services/code'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { NewTriggerModalButton } from 'components/NewTriggerModalButton/NewTriggerModalButton'
import { getErrorMessage } from 'utils/Utils'
import { useConfirmAct } from 'hooks/useConfirmAction'
import css from './PipelineTriggersTab.module.scss'

type TriggerAction = {
  name: string
  value: string
}

const branchActions: TriggerAction[] = [
  { name: 'Branch Created', value: 'branch_created' },
  { name: 'Branch Updated', value: 'branch_updated' }
]

const pullRequestActions: TriggerAction[] = [
  { name: 'Pull Request Created', value: 'pullreq_created' },
  { name: 'Pull Request Updated', value: 'pullreq_branch_updated' },
  { name: 'Pull Request Reopened', value: 'pullreq_reopened' },
  { name: 'Pull Request Closed', value: 'pullreq_closed' },
  { name: 'Pull Request Merged', value: 'pullreq_merged' }
]

const tagActions: TriggerAction[] = [
  { name: 'Tag Created', value: 'tag_created' },
  { name: 'Tag Updated', value: 'tag_updated' }
]

export const allActions: TriggerAction[][] = [branchActions, pullRequestActions, tagActions]

interface TriggerMenuItemProps {
  name: string
  lastUpdated: number
  index: number
  setSelectedTrigger: (trigger: number) => void
  isSelected?: boolean
}

const TriggerMenuItem = ({ name, lastUpdated, setSelectedTrigger, index, isSelected }: TriggerMenuItemProps) => {
  return (
    <Layout.Horizontal
      spacing={'medium'}
      className={cx(css.generalContainer, css.triggerMenuItem, { [css.selected]: isSelected })}
      flex
      padding={'large'}
      onClick={() => setSelectedTrigger(index)}>
      <Layout.Vertical spacing={'small'}>
        <Text className={css.triggerName}>{name}</Text>
        <Text className={css.triggerDate}>{`Last update: ${new Date(lastUpdated).toLocaleDateString('en-US', {
          month: 'short', // abbreviated month name
          day: '2-digit', // two-digit day
          year: 'numeric' // four-digit year
        })}`}</Text>
      </Layout.Vertical>
      <FlexExpander />
      <Layout.Horizontal
        spacing={'xsmall'}
        style={{ alignItems: 'center', borderRadius: '4px' }}
        className={css.pillContainer}>
        <Text className={css.pillText} font={{ size: 'xsmall' }}>
          Internal
        </Text>
      </Layout.Horizontal>
    </Layout.Horizontal>
  )
}

interface TriggerDetailsProps {
  name: string
  repoPath: string
  pipeline: string
  refetchTriggers: () => void
  setSelectedTrigger: (trigger: number) => void
  initialDisabled: boolean
  initialActions: EnumTriggerAction[]
}

export interface TriggerFormData {
  name: string
  disabled: boolean
  actions: EnumTriggerAction[]
}

const TriggerDetails = ({
  name,
  repoPath,
  pipeline,
  refetchTriggers,
  setSelectedTrigger,
  initialActions,
  initialDisabled
}: TriggerDetailsProps) => {
  const { getString } = useStrings()
  const { showError, showSuccess, clear: clearToaster } = useToaster()

  const { mutate: updateTrigger, loading } = useMutate<TypesTrigger>({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoPath}/+/pipelines/${pipeline}/triggers/${name}`
  })
  const { mutate: deleteTrigger } = useMutate<TypesTrigger>({
    verb: 'DELETE',
    path: `/api/v1/repos/${repoPath}/+/pipelines/${pipeline}/triggers/${name}`
  })

  const confirmDeleteTrigger = useConfirmAct()

  const handleSubmit = async (formData: TriggerFormData) => {
    try {
      const payload: OpenapiUpdateTriggerRequest = {
        identifier: formData.name,
        actions: formData.actions,
        disabled: formData.disabled
      }
      await updateTrigger(payload)
      clearToaster()
      showSuccess(getString('triggers.updateSuccess'))
      refetchTriggers()
    } catch (exception) {
      clearToaster()
      showError(getErrorMessage(exception), 0, getString('triggers.failedToUpdate'))
    }
  }

  const formInitialValues: TriggerFormData = {
    name: name,
    actions: initialActions,
    disabled: initialDisabled
  }

  if (loading) {
    return <LoadingSpinner visible={true} />
  }

  return (
    <Layout.Vertical className={cx(css.generalContainer, css.editTriggerContainer)} padding={'large'}>
      <Formik
        initialValues={formInitialValues}
        formName="editTrigger"
        enableReinitialize={true}
        validationSchema={yup.object().shape({
          name: yup
            .string()
            .trim()
            .required()
            .matches(/^[a-zA-Z_][a-zA-Z0-9-_.]*$/, getString('validation.nameLogic')),
          actions: yup.array().of(yup.string()),
          disabled: yup.boolean()
        })}
        validateOnChange
        validateOnBlur
        onSubmit={handleSubmit}>
        {formik => (
          <FormikForm>
            <Layout.Horizontal padding={{ top: 'medium', left: 'large', right: 'large' }}>
              <FormInput.Text
                name="name"
                className={css.textContainer}
                label={
                  <Text color={Color.GREY_800} font={{ size: 'small' }}>
                    {getString('name')}
                  </Text>
                }
              />
              <FlexExpander />
              <Layout.Horizontal
                spacing={'xsmall'}
                style={{ alignItems: 'center', borderRadius: '4px' }}
                className={css.pillContainer}>
                <Text className={css.pillText} font={{ size: 'xsmall' }}>
                  Internal
                </Text>
              </Layout.Horizontal>
            </Layout.Horizontal>
            <Container padding={'large'}>
              <Checkbox
                name="disabled"
                label={getString('triggers.disableTrigger')}
                checked={formik.values.disabled}
                onChange={event => {
                  if (event.currentTarget.checked) {
                    formik.setFieldValue('disabled', true)
                  } else {
                    formik.setFieldValue('disabled', false)
                  }
                }}
              />
            </Container>
            <div className={css.separator} />
            <Container>
              {allActions.map((actionGroup, index) => (
                <Container className={css.actionsContainer} padding={'large'} key={index}>
                  {actionGroup.map(action => (
                    <Checkbox
                      key={action.name}
                      name="actions"
                      label={action.name}
                      value={action.value}
                      checked={formik.values.actions.includes(action.value as EnumTriggerAction)}
                      onChange={event => {
                        if (event.currentTarget.checked) {
                          formik.setFieldValue('actions', [...formik.values.actions, action.value])
                        } else {
                          formik.setFieldValue(
                            'actions',
                            formik.values.actions.filter((value: string) => value !== action.value)
                          )
                        }
                      }}
                    />
                  ))}
                </Container>
              ))}
            </Container>
            <div className={css.separator} />
            <Layout.Horizontal
              spacing="small"
              padding={{ top: 'large', left: 'large', right: 'large' }}
              style={{ alignItems: 'center' }}>
              <Button type="submit" text={getString('save')} intent={Intent.PRIMARY} disabled={loading} />
              <Button
                text={getString('delete')}
                intent={Intent.DANGER}
                variation={ButtonVariation.SECONDARY}
                onClick={() => {
                  confirmDeleteTrigger({
                    title: getString('triggers.deleteTrigger'),
                    confirmText: getString('delete'),
                    intent: Intent.DANGER,
                    message: <String useRichText stringID="triggers.deleteTriggerConfirm" vars={{ name }} />,
                    action: async () => {
                      try {
                        await deleteTrigger(null)
                        refetchTriggers()
                        setSelectedTrigger(0)
                        showSuccess(getString('triggers.deleteTriggerSuccess', { name }))
                      } catch (e) {
                        showError(getString('triggers.deleteTriggerError'))
                      }
                    }
                  })
                }}
              />
              <FlexExpander />
              {loading && <Icon intent={Intent.PRIMARY} name="steps-spinner" size={16} />}
            </Layout.Horizontal>
          </FormikForm>
        )}
      </Formik>
    </Layout.Vertical>
  )
}

interface PipelineTriggersTabsProps {
  pipeline: string
  repoPath: string
}

const PipelineTriggersTabs = ({ repoPath, pipeline }: PipelineTriggersTabsProps) => {
  const { getString } = useStrings()

  const { data, loading, refetch } = useGet<TypesTrigger[]>({
    path: `/api/v1/repos/${repoPath}/+/pipelines/${pipeline}/triggers`,
    lazy: !repoPath || !pipeline
  })

  const [selectedTrigger, setSelectedTrigger] = React.useState<number>(0)

  if (loading) {
    return <LoadingSpinner visible={true} />
  }

  return (
    <>
      <Layout.Horizontal padding={'large'}>
        <Layout.Vertical padding={'large'}>
          <NewTriggerModalButton
            modalTitle={getString('triggers.createTrigger')}
            text={getString('triggers.newTrigger')}
            variation={ButtonVariation.PRIMARY}
            icon="plus"
            onSuccess={() => refetch()}
            repoPath={repoPath}
            pipeline={pipeline}
            width="150px"
          />
          <Layout.Vertical spacing={'large'} className={css.triggerList}>
            {data?.map((trigger, index) => (
              <TriggerMenuItem
                key={trigger.identifier}
                name={trigger.identifier as string}
                lastUpdated={trigger.updated as number}
                setSelectedTrigger={setSelectedTrigger}
                index={index}
                isSelected={selectedTrigger === index}
              />
            ))}
          </Layout.Vertical>
        </Layout.Vertical>
        {data && data?.length > 0 && (
          <>
            <div className={css.separator} />
            <Layout.Vertical padding={'large'}>
              <TriggerDetails
                name={data?.[selectedTrigger]?.identifier as string}
                repoPath={repoPath}
                pipeline={pipeline}
                refetchTriggers={refetch}
                setSelectedTrigger={setSelectedTrigger}
                initialActions={data?.[selectedTrigger]?.actions as EnumTriggerAction[]}
                initialDisabled={data?.[selectedTrigger]?.disabled as boolean}
              />
            </Layout.Vertical>
          </>
        )}
      </Layout.Horizontal>
    </>
  )
}

export default PipelineTriggersTabs
