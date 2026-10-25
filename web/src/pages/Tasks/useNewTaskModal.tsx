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
import { useMutate } from 'restful-react'
import * as yup from 'yup'
import {
  Button,
  ButtonVariation,
  Dialog,
  FormInput,
  Formik,
  FormikForm,
  Layout,
  SelectOption,
  useToaster
} from '@harnessio/uicore'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import { getErrorMessage } from 'utils/Utils'
import { TASK_PRIORITIES, TaskCreateInput, TaskPriority, TypesTask, getTaskApiPath } from 'utils/TaskUtils'

interface FormData {
  title: string
  description: string
  priority: SelectOption
}

interface UseNewTaskModalProps {
  repoPath?: string
  onCreated: (task: TypesTask) => void
}

export function useNewTaskModal({ repoPath, onCreated }: UseNewTaskModalProps) {
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const { mutate: createTask, loading } = useMutate<TypesTask>({
    verb: 'POST',
    path: getTaskApiPath(repoPath)
  })

  const [openModal, hideModal] = useModalHook(() => {
    const onSubmit = (data: FormData) => {
      const payload: TaskCreateInput = {
        title: data.title.trim(),
        description: data.description,
        priority: data.priority.value as TaskPriority
      }
      createTask(payload)
        .then(task => {
          hideModal()
          showSuccess(getString('tasks.taskCreated'))
          onCreated(task)
        })
        .catch(error => showError(getErrorMessage(error)))
    }

    return (
      <Dialog isOpen enforceFocus={false} onClose={hideModal} title={getString('tasks.newTask')}>
        <Formik<FormData>
          initialValues={{ title: '', description: '', priority: { label: 'P2', value: 'p2' } }}
          formName="createNewTask"
          validationSchema={yup.object().shape({
            title: yup
              .string()
              .trim()
              .required(`${getString('tasks.taskTitle')} ${getString('isRequired')}`)
          })}
          validateOnChange
          validateOnBlur
          onSubmit={onSubmit}>
          <FormikForm>
            <Layout.Vertical spacing="small">
              <FormInput.Text
                name="title"
                label={getString('tasks.taskTitle')}
                inputGroup={{ autoFocus: true }}
                tooltipProps={{ dataTooltipId: 'taskTitle' }}
              />
              <FormInput.TextArea name="description" label={getString('tasks.description')} />
              <FormInput.Select
                name="priority"
                label={getString('tasks.priority')}
                items={TASK_PRIORITIES.map(p => ({ label: p.toUpperCase(), value: p }))}
              />
              <Layout.Horizontal spacing="medium" padding={{ top: 'medium' }}>
                <Button
                  variation={ButtonVariation.PRIMARY}
                  text={getString('tasks.createTask')}
                  type="submit"
                  disabled={loading}
                />
                <Button variation={ButtonVariation.TERTIARY} text={getString('cancel')} onClick={hideModal} />
              </Layout.Horizontal>
            </Layout.Vertical>
          </FormikForm>
        </Formik>
      </Dialog>
    )
  }, [repoPath, loading])

  return { openModal, hideModal }
}
