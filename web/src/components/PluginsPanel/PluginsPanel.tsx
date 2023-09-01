import React, { useEffect, useState } from 'react'
import { Container, Layout, Tab, Tabs, Text } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'

import css from './PluginsPanel.module.scss'
import { Color, FontVariation } from '@harnessio/design-system'
import { Icon, type IconName } from '@harnessio/icons'

enum PluginCategory {
  Harness,
  Drone
}

enum PluginPanelView {
  Category,
  Listing
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

export const PluginsPanel = (): JSX.Element => {
  const { getString } = useStrings()
  const [category, setCategory] = useState<PluginCategory>()
  const [panelView, setPanelView] = useState<PluginPanelView>(PluginPanelView.Category)
  const [plugins, setPlugins] = useState<Record<string, any>[]>([])

  useEffect(() => {
    if (category === PluginCategory.Drone) {
      setPlugins(plugins)
    }
  }, [category])

  const renderPluginCategories = (): JSX.Element => {
    return (
      <>
        {PluginCategories.map((item: PluginInterface) => {
          const { name, category, description, icon } = item
          return (
            <Layout.Horizontal
              onClick={() => {
                setCategory(category)
                if (category === PluginCategory.Drone) {
                  setPanelView(PluginPanelView.Listing)
                }
              }}
              key={category}
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

  const renderPluginListing = (): JSX.Element => {
    return <></>
  }

  const renderPluginsPanel = (): JSX.Element => {
    switch (panelView) {
      case PluginPanelView.Category:
        return renderPluginCategories()
      case PluginPanelView.Listing:
        return renderPluginListing()
      default:
        return <></>
    }
  }

  return (
    <Container className={css.main}>
      <Tabs id={'pluginsPanel'} defaultSelectedTabId={'plugins'} className={css.tabs}>
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
