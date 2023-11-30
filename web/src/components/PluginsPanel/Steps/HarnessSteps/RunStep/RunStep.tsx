import React, { useCallback, useEffect } from 'react'
import { connect, getIn } from 'formik'
import type { FormikContextProps } from '@harnessio/uicore/dist/components/FormikForm/utils'
import { has, isString } from 'lodash-es'
import { Accordion, FormInput, Layout } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'

import css from './RunStep.module.scss'

function _RunStep({ formik }: FormikContextProps<any>): JSX.Element {
  const { getString } = useStrings()

  useEffect(() => {
    const container = getIn(formik?.values, 'container')
    if (isString(container)) {
      formik?.setFieldValue('container.image', container)
    }
  }, [formik?.values])

  const getActiveTabId = useCallback((): string => {
    if (has(formik?.values, 'container')) {
      return 'container'
    } else if (has(formik?.values, 'mount')) {
      return 'mount'
    }
    return ''
  }, [formik?.values])

  return (
    <Layout.Vertical width="inherit">
      <FormInput.Text name={'name'} label={getString('name')} style={{ width: '100%' }} key={'name'} />
      <FormInput.TextArea
        name={'script'}
        label={getString('pluginsPanel.run.script')}
        style={{ width: '100%' }}
        key={'script'}
      />
      <FormInput.Select
        name={'shell'}
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
                name={'container.image'}
                label={getString('pluginsPanel.run.image')}
                style={{ width: '100%' }}
                key={'container.image'}
              />
              <FormInput.Select
                name={'container.pull'}
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
                name={'container.entrypoint'}
                label={getString('pluginsPanel.run.entrypoint')}
                style={{ width: '100%' }}
                key={'container.entrypoint'}
              />
              <FormInput.Text
                name={'container.network'}
                label={getString('pluginsPanel.run.network')}
                style={{ width: '100%' }}
                key={'container.network'}
              />
              <FormInput.Text
                name={'container.networkMode'}
                label={getString('pluginsPanel.run.networkMode')}
                style={{ width: '100%' }}
                key={'container.networkMode'}
              />
              <FormInput.Toggle
                name={'container.privileged'}
                label={getString('pluginsPanel.run.privileged')}
                style={{ width: '100%' }}
                key={'container.privileged'}
              />
              <FormInput.Text
                name={'container.user'}
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
                        name={'container.credentials.username'}
                        label={getString('pluginsPanel.run.username')}
                        style={{ width: '100%' }}
                        key={'container.credentials.username'}
                      />
                      <FormInput.Text
                        name={'container.credentials.password'}
                        label={getString('pluginsPanel.run.password')}
                        style={{ width: '100%' }}
                        key={'container.credentials.password'}
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
                name={'mount.name'}
                label={getString('name')}
                style={{ width: '100%' }}
                key={'mount.name'}
              />
              <FormInput.Text
                name={'mount.path'}
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
