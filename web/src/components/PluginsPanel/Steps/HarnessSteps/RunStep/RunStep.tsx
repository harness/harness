import React, { useCallback, useEffect } from 'react'
import { connect, getIn, useFormikContext } from 'formik'
import type { FormikContextProps } from '@harnessio/uicore/dist/components/FormikForm/utils'
import { has, isEmpty, isString } from 'lodash-es'
import { Accordion, FormInput, Layout } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import type { PluginFormDataInterface } from 'components/PluginsPanel/PluginsPanel'

import css from './RunStep.module.scss'

interface RunStepProps extends FormikContextProps<PluginFormDataInterface> {
  prefix: string
}

function _RunStep({ formik, prefix }: RunStepProps): JSX.Element {
  const { setFieldValue, initialValues } = useFormikContext<PluginFormDataInterface>()
  const { getString } = useStrings()

  useEffect(() => {
    if (!isEmpty(initialValues)) {
      const prefixWithSpec = prefix ? `${prefix}.spec` : 'spec'
      const containerImage = getIn(initialValues, `${prefixWithSpec}.container`)
      if (isString(containerImage)) {
        setFieldValue(`${prefixWithSpec}.container.image`, containerImage)
      }
    }
  }, [initialValues]) // eslint-disable-line react-hooks/exhaustive-deps

  const getActiveTabId = useCallback((): string => {
    if (has(initialValues, getSpecFieldNameWithPrefix('container'))) {
      return 'container'
    } else if (has(initialValues, getSpecFieldNameWithPrefix('mount'))) {
      return 'mount'
    }
    return ''
  }, [initialValues])

  const getFieldNameWithPrefix = useCallback(
    (fieldName: string) => {
      return prefix ? `${prefix}.${fieldName}` : fieldName
    },
    [prefix]
  )

  const getSpecFieldNameWithPrefix = useCallback(
    (fieldName: string) => {
      const prefixWithSpec = prefix ? `${prefix}.spec` : 'spec'
      return prefix ? `${prefixWithSpec}.${fieldName}` : fieldName
    },
    [prefix]
  )

  return (
    <Layout.Vertical width="inherit">
      {/* "name" gets rendered at outside step's spec */}
      <FormInput.Text
        name={getFieldNameWithPrefix('name')}
        label={getString('name')}
        style={{ width: '100%' }}
        key={'name'}
      />
      <FormInput.TextArea
        name={getSpecFieldNameWithPrefix('script')}
        label={getString('pluginsPanel.run.script')}
        style={{ width: '100%' }}
        key={'script'}
      />
      <FormInput.Select
        name={getSpecFieldNameWithPrefix('shell')}
        label={getString('pluginsPanel.run.shell')}
        style={{ width: '100%' }}
        key={'shell'}
        items={[
          { label: getString('pluginsPanel.run.sh'), value: 'sh' },
          { label: getString('pluginsPanel.run.bash'), value: 'bash' },
          { label: getString('pluginsPanel.run.powershell'), value: 'powershell' },
          { label: getString('pluginsPanel.run.pwsh'), value: 'pwsh' }
        ]}
      />
      <Accordion activeId={getActiveTabId()}>
        <Accordion.Panel
          id="container"
          summary="Container"
          details={
            <Layout.Vertical className={css.indent}>
              <FormInput.Text
                name={getSpecFieldNameWithPrefix('container.image')}
                label={getString('pluginsPanel.run.image')}
                style={{ width: '100%' }}
                key={'container.image'}
              />
              <FormInput.Select
                name={getSpecFieldNameWithPrefix('container.pull')}
                label={getString('pluginsPanel.run.pull')}
                style={{ width: '100%' }}
                key={'container.pull'}
                items={[
                  { label: getString('pluginsPanel.run.always'), value: 'always' },
                  { label: getString('pluginsPanel.run.never'), value: 'never' },
                  { label: getString('pluginsPanel.run.ifNotExists'), value: 'if-not-exists' }
                ]}
              />
              <FormInput.Text
                name={getSpecFieldNameWithPrefix('container.entrypoint')}
                label={getString('pluginsPanel.run.entrypoint')}
                style={{ width: '100%' }}
                key={'container.entrypoint'}
              />
              <FormInput.Text
                name={getSpecFieldNameWithPrefix('container.network')}
                label={getString('pluginsPanel.run.network')}
                style={{ width: '100%' }}
                key={'container.network'}
              />
              <FormInput.Text
                name={getSpecFieldNameWithPrefix('container.networkMode')}
                label={getString('pluginsPanel.run.networkMode')}
                style={{ width: '100%' }}
                key={'container.networkMode'}
              />
              <FormInput.Toggle
                name={getSpecFieldNameWithPrefix('container.privileged')}
                label={getString('pluginsPanel.run.privileged')}
                style={{ width: '100%' }}
                key={'container.privileged'}
              />
              <FormInput.Text
                name={getSpecFieldNameWithPrefix('container.user')}
                label={getString('user')}
                style={{ width: '100%' }}
                key={'container.user'}
              />
              <Accordion activeId={has(formik?.values, 'container.credentials') ? 'container.credentials' : ''}>
                <Accordion.Panel
                  id="container.credentials"
                  summary={getString('pluginsPanel.run.credentials')}
                  details={
                    <Layout.Vertical className={css.indent}>
                      <FormInput.Text
                        name={getSpecFieldNameWithPrefix('container.credentials.username')}
                        label={getString('pluginsPanel.run.username')}
                        style={{ width: '100%' }}
                        key={getSpecFieldNameWithPrefix('container.credentials.username')}
                      />
                      <FormInput.Text
                        name={getSpecFieldNameWithPrefix('container.credentials.password')}
                        label={getString('pluginsPanel.run.password')}
                        style={{ width: '100%' }}
                        key={getSpecFieldNameWithPrefix('container.credentials.password')}
                      />
                    </Layout.Vertical>
                  }
                />
              </Accordion>
            </Layout.Vertical>
          }
        />
        <Accordion.Panel
          id="mount"
          summary="Mount"
          details={
            <Layout.Vertical className={css.indent}>
              <FormInput.Text
                name={getSpecFieldNameWithPrefix('mount.name')}
                label={getString('name')}
                style={{ width: '100%' }}
                key={'mount.name'}
              />
              <FormInput.Text
                name={getSpecFieldNameWithPrefix('mount.path')}
                label={getString('pluginsPanel.run.path')}
                style={{ width: '100%' }}
                key={'mount.path'}
              />
            </Layout.Vertical>
          }
        />
      </Accordion>
    </Layout.Vertical>
  )
}

export const RunStep = connect(_RunStep)
