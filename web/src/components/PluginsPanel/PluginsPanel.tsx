import React, { useCallback, useEffect, useState } from 'react'
import { Formik } from 'formik'
import { parse } from 'yaml'
import { capitalize, get, omit, set } from 'lodash-es'
import { Classes, PopoverInteractionKind, PopoverPosition } from '@blueprintjs/core'
import type { TypesPlugin } from 'services/code'
import { Color, FontVariation } from '@harnessio/design-system'
import { Icon, type IconName } from '@harnessio/icons'
import {
  Accordion,
  Button,
  ButtonVariation,
  Container,
  ExpandingSearchInput,
  FormInput,
  FormikForm,
  Layout,
  Popover,
  Text
} from '@harnessio/uicore'
import { useStrings } from 'framework/strings'

import css from './PluginsPanel.module.scss'

enum PluginCategory {
  Harness,
  Drone
}

enum PluginPanelView {
  Category,
  Listing,
  Configuration
}

interface PluginInput {
  type: 'string'
  description?: string
  default?: string
  options?: { isExtended?: boolean }
}

interface PluginCategoryInterface {
  category: PluginCategory
  name: string
  description: string
  icon: IconName
}

const RunStep: TypesPlugin = {
  uid: 'run',
  description: 'Run a script',
  spec: '{"kind":"run","type":"step","name":"Run","spec":{"name":"run","description":"Run a script","inputs":{"image":{"type":"string","description":"Container image","required":true},"script":{"type":"string","description":"Script to execute","required":true,"options":{"isExtended":true}}}}}'
}

interface PluginInsertionTemplateInterface {
  name?: string
  type: 'plugin'
  spec: {
    name: string
    inputs: { [key: string]: string }
  }
}

const PluginInsertionTemplate: PluginInsertionTemplateInterface = {
  name: '<step-name>',
  type: 'plugin',
  spec: {
    name: '<plugin-uid-from-database>',
    inputs: {
      '<param1>': '<value1>',
      '<param2>': '<value2>'
    }
  }
}

const PluginNameFieldPath = 'spec.name'
const PluginInputsFieldPath = 'spec.inputs'

const LIST_FETCHING_LIMIT = 100

export interface PluginsPanelInterface {
  onPluginAddUpdate: (isUpdate: boolean, pluginFormData: Record<string, any>) => void
}

export const PluginsPanel = ({ onPluginAddUpdate }: PluginsPanelInterface): JSX.Element => {
  const { getString } = useStrings()
  const [category, setCategory] = useState<PluginCategory>()
  const [panelView, setPanelView] = useState<PluginPanelView>(PluginPanelView.Category)
  const [plugin, setPlugin] = useState<TypesPlugin>()
  const [plugins, setPlugins] = useState<TypesPlugin[]>([])
  const [query, setQuery] = useState<string>('')
  const [loading, setLoading] = useState<boolean>(false)

  const PluginCategories: PluginCategoryInterface[] = [
    {
      category: PluginCategory.Harness,
      name: 'Run',
      description: getString('pluginsPanel.run.helptext'),
      icon: 'run-step'
    },
    {
      category: PluginCategory.Drone,
      name: 'Plugin',
      description: getString('pluginsPanel.plugins.helptext'),
      icon: 'ci-infra'
    }
  ]

  const fetchAllPlugins = useCallback((): void => {
    try {
      setLoading(true)
      let allPlugins: TypesPlugin[] = []
      fetch(`/api/v1/plugins?page=${1}&limit=${LIST_FETCHING_LIMIT}`)
        .then(async response => {
          const plugins = await response.json()
          allPlugins = [...plugins]
          fetch(`/api/v1/plugins?page=${2}&limit=${LIST_FETCHING_LIMIT}`).then(async response => {
            const plugins = await response.json()
            setPlugins([...allPlugins, ...plugins])
          })
        })
        .catch(_err => {
          /* ignore error */
        })
        .catch(_err => {
          /* ignore error */
        })
      setLoading(false)
    } catch (ex) {
      /* ignore exception */
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (category === PluginCategory.Drone) {
      fetchAllPlugins()
    }
  }, [category])

  useEffect(() => {
    if (panelView === PluginPanelView.Listing) {
      if (query) {
        setPlugins(existingPlugins => existingPlugins.filter((item: TypesPlugin) => item.uid?.includes(query)))
      } else {
        fetchAllPlugins()
      }
    }
  }, [query])

  const renderPluginCategories = (): JSX.Element => {
    return (
      <>
        {PluginCategories.map((item: PluginCategoryInterface) => {
          const { name, category: pluginCategory, description, icon } = item
          return (
            <Layout.Horizontal
              onClick={() => {
                setCategory(pluginCategory)
                if (pluginCategory === PluginCategory.Drone) {
                  setPanelView(PluginPanelView.Listing)
                } else if (pluginCategory === PluginCategory.Harness) {
                  setPlugin(RunStep)
                  setPanelView(PluginPanelView.Configuration)
                }
              }}
              key={pluginCategory}
              padding={{ left: 'medium', right: 'medium', top: 'medium', bottom: 'medium' }}
              flex={{ justifyContent: 'flex-start' }}
              className={css.plugin}>
              <Container padding="small" className={css.pluginIcon}>
                <Icon name={icon} />
              </Container>
              <Layout.Vertical padding={{ left: 'small' }}>
                <Text color={Color.PRIMARY_7} font={{ variation: FontVariation.BODY2 }}>
                  {name}
                </Text>
                <Text font={{ variation: FontVariation.SMALL }}>{description}</Text>
              </Layout.Vertical>
            </Layout.Horizontal>
          )
        })}
      </>
    )
  }

  const renderPlugins = useCallback((): JSX.Element => {
    return loading ? (
      <Container flex={{ justifyContent: 'center' }} padding="large">
        <Icon name="steps-spinner" color={Color.PRIMARY_7} size={25} />
      </Container>
    ) : (
      <Layout.Vertical spacing="small" padding={{ top: 'small' }}>
        <Layout.Horizontal flex={{ justifyContent: 'space-between' }} padding={{ left: 'small', right: 'xlarge' }}>
          <Layout.Horizontal
            flex={{ justifyContent: 'flex-start', alignItems: 'center' }}
            spacing="small"
            padding={{ top: 'medium', bottom: 'medium', left: 'medium' }}>
            <Icon
              name="arrow-left"
              size={18}
              onClick={() => {
                setPanelView(PluginPanelView.Category)
              }}
              className={css.arrow}
            />
            <Text font={{ variation: FontVariation.H5 }}>{getString('plugins.select')}</Text>
          </Layout.Horizontal>
          <ExpandingSearchInput
            autoFocus={true}
            alwaysExpanded={true}
            defaultValue={query}
            onChange={setQuery}
            className={css.search}
          />
        </Layout.Horizontal>
        <Container className={css.plugins}>
          {plugins?.map((pluginItem: TypesPlugin) => {
            const { uid, description } = pluginItem
            return (
              <Layout.Horizontal
                flex={{ justifyContent: 'flex-start' }}
                padding={{ left: 'large', top: 'medium', bottom: 'medium', right: 'large' }}
                className={css.plugin}
                onClick={() => {
                  setPanelView(PluginPanelView.Configuration)
                  setPlugin(pluginItem)
                }}
                key={uid}>
                <Icon name={'gear'} size={25} />
                <Layout.Vertical padding={{ left: 'small' }}>
                  <Text font={{ variation: FontVariation.BODY2 }} color={Color.PRIMARY_7}>
                    {uid}
                  </Text>
                  <Text font={{ variation: FontVariation.SMALL }} className={css.pluginDesc}>
                    {description}
                  </Text>
                </Layout.Vertical>
              </Layout.Horizontal>
            )
          })}
        </Container>
      </Layout.Vertical>
    )
  }, [loading, plugins, query])

  const generateFriendlyName = useCallback((pluginName: string): string => {
    return capitalize(pluginName.split('_').join(' '))
  }, [])

  const generateLabelForPluginField = useCallback(
    ({ name, properties }: { name: string; properties: PluginInput }): JSX.Element | string => {
      const { description } = properties
      return (
        <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
          {name && <Text font={{ variation: FontVariation.FORM_LABEL }}>{generateFriendlyName(name)}</Text>}
          {description && (
            <Popover
              interactionKind={PopoverInteractionKind.HOVER}
              boundary="viewport"
              position={PopoverPosition.RIGHT}
              popoverClassName={Classes.DARK}
              content={
                <Container padding="medium">
                  <Text font={{ variation: FontVariation.SMALL }} color={Color.WHITE}>
                    {description}
                  </Text>
                </Container>
              }>
              <Icon name="info" color={Color.PRIMARY_7} size={10} padding={{ bottom: 'small' }} />
            </Popover>
          )}
        </Layout.Horizontal>
      )
    },
    []
  )

  const renderPluginFormField = ({ name, properties }: { name: string; properties: PluginInput }): JSX.Element => {
    const { type, options } = properties
    const { isExtended } = options || {}
    const WrapperComponent = isExtended ? FormInput.TextArea : FormInput.Text
    return type === 'string' ? (
      <WrapperComponent
        name={name}
        label={generateLabelForPluginField({ name, properties })}
        style={{ width: '100%' }}
        key={name}
      />
    ) : (
      <></>
    )
  }

  const constructPayloadForYAMLInsertion = (
    pluginFormData: Record<string, any>,
    pluginMetadata?: TypesPlugin
  ): Record<string, any> => {
    const { name } = pluginFormData
    switch (category) {
      case PluginCategory.Drone:
        let payload = { ...PluginInsertionTemplate }
        /* Step name is optional, set only if specified by user */
        if (name) {
          set(payload, 'name', name)
        } else {
          payload = omit(payload, 'name')
        }
        set(payload, PluginNameFieldPath, pluginMetadata?.uid)
        set(payload, PluginInputsFieldPath, omit(pluginFormData, 'name'))
        return payload as PluginInsertionTemplateInterface
      case PluginCategory.Harness:
        return {
          ...(name && { name }),
          type: 'run',
          spec: pluginFormData
        }
      default:
        return {}
    }
  }

  const insertNameFieldToPluginInputs = (existingInputs: {
    [key: string]: PluginInput
  }): { [key: string]: PluginInput } => {
    const inputsClone = Object.assign(
      {
        name: {
          type: 'string',
          description: 'Name of the step'
        }
      },
      existingInputs
    )
    return inputsClone
  }

  const getPluginInputsFromSpec = useCallback((pluginSpec: string): Record<string, any> => {
    if (!pluginSpec) {
      return {}
    }
    try {
      const pluginSpecAsObj = parse(pluginSpec)
      return get(pluginSpecAsObj, 'spec.inputs', {})
    } catch (ex) {}
    return {}
  }, [])

  const renderPluginConfigForm = useCallback((): JSX.Element => {
    const pluginInputs = getPluginInputsFromSpec(get(plugin, 'spec', '') as string)
    if (category === PluginCategory.Drone && Object.keys(pluginInputs).length === 0) {
      return <></>
    }
    const allPluginInputs = insertNameFieldToPluginInputs(pluginInputs)
    return (
      <Layout.Vertical spacing="large" className={css.configForm}>
        <Layout.Horizontal spacing="small" flex={{ justifyContent: 'flex-start' }}>
          <Icon
            name="arrow-left"
            size={18}
            onClick={() => {
              setPlugin(undefined)
              if (category === PluginCategory.Drone) {
                setPanelView(PluginPanelView.Listing)
              } else if (category === PluginCategory.Harness) {
                setPanelView(PluginPanelView.Category)
              }
            }}
            className={css.arrow}
          />
          {plugin?.uid && (
            <Text font={{ variation: FontVariation.H5 }}>
              {getString('addLabel')} {plugin.uid} {getString('plugins.stepLabel')}
            </Text>
          )}
        </Layout.Horizontal>
        <Container className={css.form}>
          <Formik
            initialValues={{}}
            onSubmit={formData => {
              onPluginAddUpdate?.(false, constructPayloadForYAMLInsertion(formData, plugin))
            }}>
            <FormikForm height="100%" flex={{ justifyContent: 'space-between', alignItems: 'baseline' }}>
              <Layout.Vertical flex={{ alignItems: 'flex-start' }} height="inherit" spacing="medium">
                <Layout.Vertical
                  width="100%"
                  className={css.formFields}
                  spacing="xsmall"
                  flex={{ justifyContent: 'space-between' }}>
                  {category === PluginCategory.Harness ? (
                    <Layout.Vertical width="inherit">
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
                      <Accordion activeId="container">
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
                              <Accordion activeId="container.credentials">
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
                              <FormInput.Text
                                name={'container.pull'}
                                label={getString('pluginsPanel.run.pull')}
                                style={{ width: '100%' }}
                                key={'container.pull'}
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
                              <FormInput.RadioGroup
                                name={'container.privileged'}
                                label={getString('pluginsPanel.run.privileged')}
                                style={{ width: '100%' }}
                                key={'container.privileged'}
                                items={[
                                  { label: 'Yes', value: 'true' },
                                  { label: 'No', value: 'false' }
                                ]}
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
                  ) : (
                    <Layout.Vertical width="inherit">
                      {Object.keys(allPluginInputs).map((field: string) => {
                        return renderPluginFormField({ name: field, properties: get(allPluginInputs, field) })
                      })}
                    </Layout.Vertical>
                  )}
                </Layout.Vertical>
                <Container margin={{ top: 'small', bottom: 'small' }}>
                  <Button variation={ButtonVariation.PRIMARY} text={getString('addLabel')} type="submit" />
                </Container>
              </Layout.Vertical>
            </FormikForm>
          </Formik>
        </Container>
      </Layout.Vertical>
    )
  }, [plugin, category])

  const renderPluginsPanel = useCallback((): JSX.Element => {
    switch (panelView) {
      case PluginPanelView.Category:
        return renderPluginCategories()
      case PluginPanelView.Listing:
        return renderPlugins()
      case PluginPanelView.Configuration:
        return renderPluginConfigForm()
      default:
        return <></>
    }
  }, [loading, plugins, panelView, category])

  return (
    <Layout.Vertical height="100%">
      {panelView === PluginPanelView.Category ? (
        <Container padding={{ top: 'medium', bottom: 'medium', left: 'medium' }}>
          <Text font={{ variation: FontVariation.H5 }}>{getString('step.select')}</Text>
        </Container>
      ) : (
        <></>
      )}
      {renderPluginsPanel()}
    </Layout.Vertical>
  )
}
