import React, { useCallback, useEffect, useState } from 'react'
import { Formik } from 'formik'
import { capitalize, get } from 'lodash-es'
import { Classes, PopoverInteractionKind, PopoverPosition } from '@blueprintjs/core'
import { Color, FontVariation } from '@harnessio/design-system'
import { Icon, type IconName } from '@harnessio/icons'
import { Button, ButtonVariation, Container, FormInput, FormikForm, Layout, Popover, Text } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'

import pluginList from './plugins/plugins.json'

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

interface Plugin {
  name: string
  spec: {
    name: string
    description?: string
    inputs: { [key: string]: PluginInput }
  }
}

interface PluginCategoryInterface {
  category: PluginCategory
  name: string
  description: string
  icon: IconName
}

const PluginCategories: PluginCategoryInterface[] = [
  {
    category: PluginCategory.Harness,
    name: 'Run',
    description: 'Run a script on macOS, Linux, or Windows',
    icon: 'run-step'
  },
  { category: PluginCategory.Drone, name: 'Drone', description: 'Run Drone plugins', icon: 'ci-infra' }
]

const RunStep: Plugin = {
  name: 'run',
  spec: {
    name: 'Run',
    inputs: {
      script: {
        type: 'string',
        options: { isExtended: true }
      }
    }
  }
}

export interface PluginsPanelInterface {
  onPluginAddUpdate: (isUpdate: boolean, pluginFormData: Record<string, any>) => void
}

export const PluginsPanel = ({ onPluginAddUpdate }: PluginsPanelInterface): JSX.Element => {
  const { getString } = useStrings()
  const [category, setCategory] = useState<PluginCategory>()
  const [panelView, setPanelView] = useState<PluginPanelView>(PluginPanelView.Category)
  const [plugin, setPlugin] = useState<Plugin>()
  const [plugins, setPlugins] = useState<Plugin[]>()
  const [loading] = useState<boolean>(false)

  const fetchPlugins = () => {
    /* temporarily done till api response gets available */
    //@ts-ignore
    setPlugins(pluginList)
  }

  useEffect(() => {
    if (category === PluginCategory.Drone) {
      fetchPlugins()
    }
  }, [category])

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
        </Layout.Horizontal>
        <Container className={css.plugins}>
          {plugins?.map((pluginItem: Plugin) => {
            const { name: uid, description } = pluginItem.spec
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
  }, [loading, plugins])

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
    const { type, default: defaultValue, options } = properties
    const { isExtended } = options || {}
    const WrapperComponent = isExtended ? FormInput.TextArea : FormInput.Text
    return type === 'string' ? (
      <WrapperComponent
        name={name}
        label={generateLabelForPluginField({ name, properties })}
        style={{ width: '100%' }}
        key={name}
        placeholder={defaultValue}
      />
    ) : (
      <></>
    )
  }

  const constructPayloadForYAMLInsertion = (pluginFormData: Record<string, any>): Record<string, any> => {
    let constructedPayload = { ...pluginFormData }
    switch (category) {
      case PluginCategory.Drone:
        return { type: 'script', spec: constructedPayload }
      case PluginCategory.Harness:
        return { type: 'script', spec: { run: get(constructedPayload, 'script', '') } }
      default:
        return {}
    }
  }

  const renderPluginConfigForm = useCallback((): JSX.Element => {
    const inputs: { [key: string]: PluginInput } = get(plugin, 'spec.inputs', {})
    return (
      <Layout.Vertical
        spacing="large"
        padding={{ left: 'xxlarge', top: 'large', right: 'xxlarge', bottom: 'xxlarge' }}
        className={css.panelContent}>
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
          {plugin?.spec?.name && (
            <Text font={{ variation: FontVariation.H5 }}>
              {getString('addLabel')} {plugin.spec.name} {getString('plugins.stepLabel')}
            </Text>
          )}
        </Layout.Horizontal>
        <Container className={css.form}>
          <Formik
            initialValues={{}}
            onSubmit={formData => {
              onPluginAddUpdate?.(false, constructPayloadForYAMLInsertion(formData))
            }}>
            <FormikForm height="100%" flex={{ justifyContent: 'space-between', alignItems: 'baseline' }}>
              <Layout.Vertical flex={{ alignItems: 'flex-start' }} height="100%">
                {inputs && (
                  <Layout.Vertical width="100%" className={css.formFields} spacing="xsmall">
                    {Object.keys(inputs).map((field: string) => {
                      return renderPluginFormField({ name: field, properties: get(inputs, field) })
                    })}
                  </Layout.Vertical>
                )}
                <Button variation={ButtonVariation.PRIMARY} text={getString('addLabel')} type="submit" />
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
