import React, { useCallback, useEffect, useState } from 'react'
import cx from 'classnames'
import { Card, Container, Layout, Text } from '@harnessio/uicore'
import { Icon, type IconProps } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { EntityAddUpdateInterface, PluginsPanel } from 'components/PluginsPanel/PluginsPanel'
import { PipelineEntity } from './types'

import css from './PipelineConfigPanel.module.scss'

export interface PipelineConfigPanelProps {
  entityDataFromYAML: EntityAddUpdateInterface
  onEntityAddUpdate: (data: EntityAddUpdateInterface) => void
  entityFieldUpdateData: Partial<EntityAddUpdateInterface>
}

interface ConfigPanelEntityInterface {
  entity: PipelineEntity
  name: string
  description: string
  icon: IconProps
}

const PipelineConfigPanel: React.FC<PipelineConfigPanelProps> = props => {
  const { entityDataFromYAML, onEntityAddUpdate, entityFieldUpdateData } = props
  const { getString } = useStrings()
  const [selectedEntity, setSelectedEntity] = useState<PipelineEntity | undefined>()

  useEffect(() => {
    setSelectedEntity(entityDataFromYAML?.entity)
  }, [entityDataFromYAML?.entity])

  const ConfigPanelEntities: ConfigPanelEntityInterface[] = [
    { entity: PipelineEntity.STEP, name: 'Step', description: 'Add a step', icon: { name: 'run-step', size: 18 } }
  ]

  const renderPanelForEntity = (entitySelected: PipelineEntity): JSX.Element => {
    switch (entitySelected) {
      case PipelineEntity.STEP:
        return (
          <PluginsPanel
            onPluginAddUpdate={onEntityAddUpdate}
            pluginDataFromYAML={entityDataFromYAML}
            pluginFieldUpdateData={entityFieldUpdateData}
          />
        )
      default:
        return <></>
    }
  }

  const renderEntityOptions = useCallback((): JSX.Element => {
    return (
      <Layout.Vertical spacing="large">
        <Text font={{ variation: FontVariation.H4 }}>{getString('pipelineConfig.label')}</Text>
        {ConfigPanelEntities.map((entityOption: ConfigPanelEntityInterface) => {
          const { entity } = entityOption
          return (
            <Card className={cx(css.entityCard, css.cursor)} onClick={() => setSelectedEntity(entity)} key={entity}>
              <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
                <Layout.Horizontal
                  flex={{ justifyContent: 'flex-start' }}
                  className={css.cursor}
                  onClick={() => setSelectedEntity(entity)}>
                  <Container className={css.entityIcon}>
                    <Icon {...entityOption.icon} />
                  </Container>
                  <Layout.Vertical padding={{ left: 'medium' }} spacing="xsmall">
                    <Text
                      color={Color.GREY_900}
                      className={css.fontWeight600}
                      font={{ variation: FontVariation.BODY2_SEMI }}>
                      {entityOption.name}
                    </Text>
                    <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
                      {entityOption.description}
                    </Text>
                  </Layout.Vertical>
                </Layout.Horizontal>
                <Container>
                  <Icon name="arrow-right" size={24} onClick={() => setSelectedEntity(entity)} className={css.cursor} />
                </Container>
              </Layout.Horizontal>
            </Card>
          )
        })}
      </Layout.Vertical>
    )
  }, [ConfigPanelEntities]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Container height="100%" padding={{ top: 'large', bottom: 'large', left: 'xlarge', right: 'xlarge' }}>
      {selectedEntity ? renderPanelForEntity(selectedEntity) : renderEntityOptions()}
    </Container>
  )
}

export default PipelineConfigPanel
