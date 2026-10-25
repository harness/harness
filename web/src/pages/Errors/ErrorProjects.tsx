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
import { useMutate } from 'restful-react'
import * as yup from 'yup'
import {
  Button,
  ButtonSize,
  ButtonVariation,
  Container,
  Dialog,
  FormInput,
  Formik,
  FormikForm,
  Layout,
  Text,
  TextInput,
  useToaster
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { isEmpty } from 'lodash-es'
import { useStrings } from 'framework/strings'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { getErrorMessage } from 'utils/Utils'
import {
  InsightsProjectCreateInput,
  InsightsProjectUpdateInput,
  TypesInsightsProject,
  getInsightsApiPath
} from 'utils/InsightsUtils'
import css from './Errors.module.scss'

interface ErrorProjectsProps {
  repoPath: string
  projects: TypesInsightsProject[]
  onChanged: () => void
}

// Settings panel of the Insights projects: DSN, key rotation, enabled flag, rate limit and sample rate.
export const ErrorProjects: React.FC<ErrorProjectsProps> = ({ repoPath, projects, onChanged }) => {
  const { getString } = useStrings()
  const [creating, setCreating] = useState(false)

  return (
    <Layout.Vertical spacing="medium">
      <Layout.Horizontal>
        <Button
          variation={ButtonVariation.PRIMARY}
          text={getString('insights.newProject')}
          icon="plus"
          onClick={() => setCreating(true)}
        />
      </Layout.Horizontal>
      {isEmpty(projects) ? (
        <NoResultCard
          forSearch={false}
          message={getString('insights.noProjects')}
          buttonText={getString('insights.newProject')}
          onButtonClick={() => setCreating(true)}
        />
      ) : (
        projects.map(project => (
          <ProjectCard key={project.id} repoPath={repoPath} project={project} onChanged={onChanged} />
        ))
      )}
      {creating && (
        <NewProjectDialog
          repoPath={repoPath}
          onClose={() => setCreating(false)}
          onCreated={() => {
            setCreating(false)
            onChanged()
          }}
        />
      )}
    </Layout.Vertical>
  )
}

interface ProjectCardProps {
  repoPath: string
  project: TypesInsightsProject
  onChanged: () => void
}

const ProjectCard: React.FC<ProjectCardProps> = ({ repoPath, project, onChanged }) => {
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const confirmAct = useConfirmAct()
  const [rateLimit, setRateLimit] = useState(String(project.rate_limit_per_minute ?? 0))
  const [sampleRate, setSampleRate] = useState(String(project.sample_rate ?? 1))
  const projectPath = `${getInsightsApiPath(repoPath)}/projects/${project.id}`

  const { mutate: updateProject, loading: updating } = useMutate<
    TypesInsightsProject,
    unknown,
    unknown,
    InsightsProjectUpdateInput,
    unknown
  >({
    verb: 'PATCH',
    path: projectPath
  })
  const { mutate: rotateKey, loading: rotating } = useMutate<TypesInsightsProject>({
    verb: 'POST',
    path: `${projectPath}/rotate-key`
  })

  const update = (input: InsightsProjectUpdateInput) => {
    updateProject(input)
      .then(() => {
        showSuccess(getString('insights.projectUpdated'))
        onChanged()
      })
      .catch(err => showError(getErrorMessage(err)))
  }

  const saveLimits = () => {
    const rate = parseInt(rateLimit, 10)
    const sample = parseFloat(sampleRate)
    if (Number.isNaN(rate) || rate < 0 || Number.isNaN(sample) || sample < 0 || sample > 1) {
      showError(getString('insights.invalidLimits'))
      return
    }
    update({ rate_limit_per_minute: rate, sample_rate: sample })
  }

  const onRotate = () =>
    confirmAct({
      title: getString('insights.rotateKey'),
      message: getString('insights.rotateKeyConfirm'),
      action: () =>
        rotateKey({})
          .then(() => {
            showSuccess(getString('insights.keyRotated'))
            onChanged()
          })
          .catch(err => showError(getErrorMessage(err)))
    })

  return (
    <Container className={css.projectCard}>
      <Layout.Horizontal spacing="medium" flex={{ alignItems: 'center' }}>
        <Text font={{ variation: FontVariation.H5 }}>{project.identifier}</Text>
        <Text color={project.enabled ? Color.GREEN_600 : Color.GREY_500}>
          {project.enabled ? getString('enabled') : getString('insights.disabled')}
        </Text>
        <div className={css.expander} />
        <Button
          size={ButtonSize.SMALL}
          variation={ButtonVariation.SECONDARY}
          text={project.enabled ? getString('insights.disable') : getString('insights.enable')}
          disabled={updating}
          onClick={() => update({ enabled: !project.enabled })}
        />
        <Button
          size={ButtonSize.SMALL}
          variation={ButtonVariation.SECONDARY}
          text={getString('insights.rotateKey')}
          disabled={rotating}
          onClick={onRotate}
        />
      </Layout.Horizontal>

      <div className={css.field}>
        <Text color={Color.GREY_500}>{getString('insights.dsn')}</Text>
        <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
          <Text className={css.dsn}>{project.dsn}</Text>
          <CopyButton content={project.dsn || ''} tooltip={getString('insights.copyDsn')} />
        </Layout.Horizontal>
      </div>

      <Layout.Horizontal spacing="medium" flex={{ alignItems: 'flex-end', justifyContent: 'flex-start' }}>
        <div>
          <Text color={Color.GREY_500}>{getString('insights.rateLimit')}</Text>
          <TextInput value={rateLimit} onChange={e => setRateLimit((e.target as HTMLInputElement).value)} />
        </div>
        <div>
          <Text color={Color.GREY_500}>{getString('insights.sampleRate')}</Text>
          <TextInput value={sampleRate} onChange={e => setSampleRate((e.target as HTMLInputElement).value)} />
        </div>
        <Button
          variation={ButtonVariation.PRIMARY}
          size={ButtonSize.SMALL}
          text={getString('save')}
          disabled={updating}
          onClick={saveLimits}
        />
      </Layout.Horizontal>
    </Container>
  )
}

interface NewProjectFormData {
  identifier: string
  rateLimit: string
  sampleRate: string
}

const NewProjectDialog: React.FC<{ repoPath: string; onClose: () => void; onCreated: () => void }> = ({
  repoPath,
  onClose,
  onCreated
}) => {
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const { mutate: createProject, loading } = useMutate<TypesInsightsProject>({
    verb: 'POST',
    path: `${getInsightsApiPath(repoPath)}/projects`
  })

  const onSubmit = (data: NewProjectFormData) => {
    const payload: InsightsProjectCreateInput = { identifier: data.identifier.trim() }
    if (data.rateLimit.trim()) {
      payload.rate_limit_per_minute = parseInt(data.rateLimit, 10)
    }
    if (data.sampleRate.trim()) {
      payload.sample_rate = parseFloat(data.sampleRate)
    }
    createProject(payload)
      .then(() => {
        showSuccess(getString('insights.projectCreated'))
        onCreated()
      })
      .catch(err => showError(getErrorMessage(err)))
  }

  return (
    <Dialog isOpen enforceFocus={false} onClose={onClose} title={getString('insights.newProject')}>
      <Formik<NewProjectFormData>
        initialValues={{ identifier: '', rateLimit: '', sampleRate: '' }}
        formName="createInsightsProject"
        validationSchema={yup.object().shape({
          identifier: yup
            .string()
            .trim()
            .required(`${getString('insights.identifier')} ${getString('isRequired')}`),
          rateLimit: yup.number().integer().min(0),
          sampleRate: yup.number().min(0).max(1)
        })}
        validateOnChange
        validateOnBlur
        onSubmit={onSubmit}>
        <FormikForm>
          <Layout.Vertical spacing="small">
            <FormInput.Text
              name="identifier"
              label={getString('insights.identifier')}
              inputGroup={{ autoFocus: true }}
            />
            <FormInput.Text name="rateLimit" label={getString('insights.rateLimit')} />
            <FormInput.Text name="sampleRate" label={getString('insights.sampleRate')} />
            <Layout.Horizontal spacing="medium" padding={{ top: 'medium' }}>
              <Button
                variation={ButtonVariation.PRIMARY}
                text={getString('insights.createProject')}
                type="submit"
                disabled={loading}
              />
              <Button variation={ButtonVariation.TERTIARY} text={getString('cancel')} onClick={onClose} />
            </Layout.Horizontal>
          </Layout.Vertical>
        </FormikForm>
      </Formik>
    </Dialog>
  )
}
