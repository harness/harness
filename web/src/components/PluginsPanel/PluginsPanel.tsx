import React, { useCallback, useEffect, useState } from 'react'
import { Formik } from 'formik'
import { capitalize, get } from 'lodash-es'
import { useGet } from 'restful-react'
import { Color, FontVariation } from '@harnessio/design-system'
import { Icon, type IconName } from '@harnessio/icons'
import { Button, ButtonVariation, Container, FormInput, FormikForm, Layout, Tab, Tabs, Text } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import { LIST_FETCHING_LIMIT } from 'utils/Utils'
import type { TypesPlugin } from 'services/code'
import { YamlVersion } from 'pages/AddUpdatePipeline/Constants'

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

interface PluginInterface {
  category: PluginCategory
  name: string
  description: string
  icon: IconName
}

const PluginCategories: PluginInterface[] = [
  {
    category: PluginCategory.Harness,
    name: 'Run',
    description: 'Run a script on macOS, Linux, or Windows',
    icon: 'run-step'
  },
  { category: PluginCategory.Drone, name: 'Drone', description: 'Run Drone plugins', icon: 'ci-infra' }
]

const dronePluginSpecMockData = {
  inputs: {
    channel: {
      type: 'string'
    },
    token: {
      type: 'string'
    }
  },
  steps: [
    {
      type: 'script',
      spec: {
        image: 'plugins/slack'
      },
      envs: {
        PLUGIN_CHANNEL: '<+inputs.channel>'
      }
    }
  ]
}

const runStepSpec = {
  inputs: {
    script: {
      type: 'string'
    }
  }
}

export interface PluginsPanelInterface {
  version?: YamlVersion
  onPluginAddUpdate: (isUpdate: boolean, pluginFormData: Record<string, any>) => void
}

export const PluginsPanel = ({ version = YamlVersion.V0, onPluginAddUpdate }: PluginsPanelInterface): JSX.Element => {
  const { getString } = useStrings()
  const [category, setCategory] = useState<PluginCategory>()
  const [panelView, setPanelView] = useState<PluginPanelView>(PluginPanelView.Category)
  const [plugin, setPlugin] = useState<TypesPlugin>()

  const {
    data: plugins,
    loading,
    refetch: fetchPlugins
  } = useGet<TypesPlugin[]>({
    path: `/api/v1/plugins`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page: 1
    },
    lazy: true
  })

  useEffect(() => {
    if (category === PluginCategory.Drone) {
      fetchPlugins()
    }
  }, [category])

  const renderPluginCategories = (): JSX.Element => {
    return (
      <>
        {PluginCategories.map((item: PluginInterface) => {
          const { name, category: pluginCategory, description, icon } = item
          return (
            <Layout.Horizontal
              onClick={() => {
                setCategory(pluginCategory)
                if (pluginCategory === PluginCategory.Drone) {
                  setPanelView(PluginPanelView.Listing)
                } else if (pluginCategory === PluginCategory.Harness) {
                  setPlugin({ uid: getString('run') })
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
          padding={{ left: 'medium' }}>
          <Icon
            name="arrow-left"
            size={18}
            onClick={() => {
              setPanelView(PluginPanelView.Category)
            }}
            className={css.arrow}
          />
          <Text font={{ variation: FontVariation.H5 }} flex={{ justifyContent: 'center' }}>
            {getString('plugins.addAPlugin', { category: PluginCategory[category as PluginCategory] })}
          </Text>
        </Layout.Horizontal>
        <Container>
          {plugins?.map((_plugin: TypesPlugin) => {
            const { uid, description } = _plugin
            return (
              <Layout.Horizontal
                flex={{ justifyContent: 'flex-start' }}
                padding={{ left: 'large', top: 'medium', bottom: 'medium', right: 'large' }}
                className={css.plugin}
                onClick={() => {
                  setPanelView(PluginPanelView.Configuration)
                  setPlugin(_plugin)
                }}
                key={uid}>
                <Icon name={'gear'} size={25} />
                <Layout.Vertical padding={{ left: 'small' }}>
                  <Text font={{ variation: FontVariation.BODY2 }} color={Color.PRIMARY_7}>
                    {uid}
                  </Text>
                  <Text font={{ variation: FontVariation.SMALL }}>{description}</Text>
                </Layout.Vertical>
              </Layout.Horizontal>
            )
          })}
        </Container>
      </Layout.Vertical>
    )
  }, [loading, plugins])

  const renderPluginFormField = ({ name, type }: { name: string; type: 'string' }): JSX.Element => {
    return type === 'string' ? (
      <FormInput.Text
        name={name}
        label={<Text font={{ variation: FontVariation.FORM_INPUT_TEXT }}>{capitalize(name)}</Text>}
        style={{ width: '100%' }}
        key={name}
      />
    ) : (
      <></>
    )
  }

  const constructPayloadForYAMLInsertion = (isUpdate: boolean, pluginFormData: Record<string, any>) => {
    let constructedPayload = { ...pluginFormData }
    switch (category) {
      case PluginCategory.Drone:
      case PluginCategory.Harness:
        constructedPayload =
          version === YamlVersion.V1
            ? { type: 'script', spec: constructedPayload }
            : { name: 'run step', commands: [get(constructedPayload, 'script', '')] }
    }
    onPluginAddUpdate?.(isUpdate, constructedPayload)
  }

  const renderPluginConfigForm = useCallback((): JSX.Element => {
    // TODO obtain plugin input spec by parsing YAML
    const inputs = get(category === PluginCategory.Drone ? dronePluginSpecMockData : runStepSpec, 'inputs', {})
    return (
      <Layout.Vertical
        spacing="medium"
        margin={{ left: 'xxlarge', top: 'large', right: 'xxlarge', bottom: 'xxlarge' }}
        height="95%">
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
          {plugin?.uid ? (
            <Text font={{ variation: FontVariation.H5 }}>
              {getString('addLabel')} {plugin.uid} {getString('plugins.stepLabel')}
            </Text>
          ) : (
            <></>
          )}
        </Layout.Horizontal>
        <Container className={css.form}>
          <Formik
            initialValues={{}}
            onSubmit={formData => {
              constructPayloadForYAMLInsertion(false, formData)
            }}>
            <FormikForm>
              <Layout.Vertical flex={{ alignItems: 'flex-start' }} height="100%">
                <Layout.Vertical width="100%">
                  {Object.keys(inputs).map((field: string) => {
                    const fieldType = get(inputs, `${field}.type`, '') as 'string'
                    return renderPluginFormField({ name: field, type: fieldType })
                  })}
                </Layout.Vertical>
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
    <Container className={css.main}>
      <Tabs id={'pluginsPanel'} defaultSelectedTabId={'plugins'}>
        <Tab
          panelClassName={css.mainTabPanel}
          id="plugins"
          title={
            <Text
              font={{ variation: FontVariation.BODY2 }}
              padding={{ left: 'small', bottom: 'xsmall', top: 'xsmall' }}
              color={Color.PRIMARY_7}>
              {getString('plugins.title')}
            </Text>
          }
          panel={<Container className={css.pluginDetailsPanel}>{renderPluginsPanel()}</Container>}
        />
      </Tabs>
    </Container>
  )
}
